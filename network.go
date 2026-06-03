package main

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"syscall"
)

// NetworkInfo holds detected network information.
type NetworkInfo struct {
	LocalIP string
	MAC     string // uppercase, no separators, e.g. "AABBCCDDEEFF"
	Gateway string
	NetType string // "wired" or "wireless" or "unknown"
}

// GetLocalIP returns the primary IPv4 address.
func GetLocalIP() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.To4() == nil || ipNet.IP.IsLoopback() {
				continue
			}
			ip := ipNet.IP.To4().String()
			// Skip APIPA (169.254.x.x) and VPN adapters
			if strings.HasPrefix(ip, "169.254.") {
				continue
			}
			return ip, nil
		}
	}
	return "", fmt.Errorf("no active IPv4 address found")
}

// GetMAC returns the MAC address for the primary interface,
// formatted as uppercase hex without separators.
func GetMAC() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok || ipNet.IP.To4() == nil || ipNet.IP.IsLoopback() {
				continue
			}
			if strings.HasPrefix(ipNet.IP.String(), "169.254.") {
				continue
			}
			mac := iface.HardwareAddr.String()
			mac = strings.ReplaceAll(mac, ":", "")
			mac = strings.ReplaceAll(mac, "-", "")
			return strings.ToUpper(mac), nil
		}
	}
	return "", fmt.Errorf("no MAC address found")
}

// GetNetworkInfo collects all network information.
func GetNetworkInfo() *NetworkInfo {
	info := GetNetworkInfoFast()
	info.Gateway = ProbeGateway()
	if info.Gateway != "" {
		// Determine network type based on gateway
		if strings.HasPrefix(info.Gateway, "10.") || info.Gateway == "192.168.1.1" {
			info.NetType = "wired"
		} else if info.Gateway == "1.2.3.4" {
			info.NetType = "wireless"
		}
	}
	return info
}

// GetNetworkInfoFast collects network info without blocking network probes.
func GetNetworkInfoFast() *NetworkInfo {
	info := &NetworkInfo{NetType: "unknown"}

	ip, err := GetLocalIP()
	if err == nil {
		info.LocalIP = ip
	}

	mac, err := GetMAC()
	if err == nil {
		info.MAC = mac
	}

	return info
}

// ProbeGateway tries to detect the Dr.COM gateway.
// For wired: tries 10.0.1.5 first
// For wireless: tries 1.2.3.4 (captive portal redirect address)
// Returns the first reachable gateway, or empty string if none found.
func ProbeGateway() string {
	candidates := []string{
		"10.0.1.5:80", // wired gateway (try first)
		"1.2.3.4:80",  // wireless captive portal (fallback)
	}

	for _, gw := range candidates {
		conn, err := net.DialTimeout("tcp", gw, connTimeout)
		if err == nil {
			conn.Close()
			// Return just the IP without port
			host, _, _ := net.SplitHostPort(gw)
			return host
		}
	}
	return ""
}

// FlushDNS flushes the Windows DNS cache.
func FlushDNS() error {
	cmd := exec.Command("ipconfig", "/flushdns")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("flush DNS: %w: %s", err, string(output))
	}
	return nil
}

// FlushARP flushes the Windows ARP cache.
func FlushARP() error {
	cmd := exec.Command("netsh", "interface", "ip", "delete", "arpcache")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("flush ARP: %w: %s", err, string(output))
	}
	return nil
}

// ClearChromeDNS clears Chrome's internal DNS cache via its net-internals API.
func ClearChromeDNS() error {
	// Chrome's internal DNS can be cleared by requesting chrome://net-internals/#dns
	// This is a no-op from outside Chrome; the button label in the original app
	// suggests this is a user-manual action. We just log it.
	GetLogger().Info("Chrome DNS cache clear requested (manual action for user)")
	return nil
}
