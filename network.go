package main

import (
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// NetworkInfo holds detected network information.
type NetworkInfo struct {
	LocalIP string
	IPv6    string // IPv6 address, or empty if none
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

// GetIPv6 returns the primary global unicast IPv6 address, or empty string if none.
func GetIPv6() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
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
			if !ok || ipNet.IP.To4() != nil || ipNet.IP.IsLoopback() {
				continue
			}
			// Only return global unicast addresses (2000::/3), skip link-local (fe80::)
			ip := ipNet.IP
			if ip.IsGlobalUnicast() && !ip.IsPrivate() {
				return ip.String()
			}
		}
	}
	return ""
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

	ipv6 := GetIPv6()
	if ipv6 != "" {
		info.IPv6 = ipv6
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

// CheckConnectivity tests internet reachability via TCP dial to Bing.
func CheckConnectivity() bool {
	for _, addr := range []string{"www.bing.com:80", "www.bing.com:443"} {
		conn, err := net.DialTimeout("tcp", addr, 1500*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
	}
	return false
}

// CheckInternetAccess verifies real internet connectivity via HTTP GET.
// Unlike CheckConnectivity (TCP dial), this actually fetches a page and
// checks the response status — TCP dial can be hijacked by captive portals
// and report "success" even when there's no real internet access.
//
// Uses a client that follows redirects and checks the FINAL URL — if the
// request ends up at the captive portal (10.x.x.x), it's NOT real internet.
func CheckInternetAccess() bool {
	client := &http.Client{
		Timeout: 5 * time.Second,
		// Follow redirects but track the final URL
	}

	for _, url := range []string{"http://www.baidu.com", "https://www.baidu.com"} {
		resp, err := client.Get(url)
		if err != nil {
			continue
		}
		resp.Body.Close()

		// Only accept 2xx (NOT 3xx redirects — captive portals return 302)
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			continue
		}

		// If the final URL is on the captive portal (10.x.x.x or 1.2.3.4),
		// the request was hijacked — NOT real internet
		finalHost := resp.Request.URL.Hostname()
		if strings.HasPrefix(finalHost, "10.") || finalHost == "1.2.3.4" {
			GetLogger().Debug("CheckInternetAccess: hijacked to %s — no real internet", finalHost)
			continue
		}

		return true
	}
	return false
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
