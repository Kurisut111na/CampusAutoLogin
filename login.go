package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// =============================================================================
// Dr.COM Login — Dual Engine
// =============================================================================

// Carriers maps operator codes to Dr.COM suffixes.
var Carriers = map[string]string{
	"cmcc":    "@cmcc",
	"unicom":  "@unicom",
	"telecom": "@telecom",
	"campus":  "", // campus network uses no carrier suffix
}

// CarrierNames maps operator codes to display names.
var CarrierNames = map[string]string{
	"cmcc":    "中国移动",
	"unicom":  "中国联通",
	"telecom": "中国电信",
	"campus":  "校园网",
}

// OperatorToSuffix converts an operator code to the Dr.COM suffix.
func OperatorToSuffix(op string) string {
	if suffix, ok := Carriers[op]; ok {
		return suffix
	}
	return "@cmcc"
}

// ACInfo holds BRAS information extracted from the Portal page.
type ACInfo struct {
	IP     string // AC IP (e.g. "10.32.255.10")
	Name   string // AC name (e.g. "HJ-BRAS-ME60-01")
	UserIP string // BRAS-provided user IP from redirect URL (e.g. "10.34.93.144") — more authoritative than local detection
	MAC    string // BRAS-provided MAC, no separators (e.g. "849E56C91181") — for old API
	MACRaw string // BRAS-provided MAC, original format (e.g. "84-9E-56-C9-11-81") — for Portal v4.0
	AreaID string // BRAS area ID from redirect URL (e.g. "ethtrunk/102:2757.0")
}

// LoginResult holds the result of a login attempt.
type LoginResult struct {
	Success       bool
	AlreadyOnline bool   // server says already online (may be zombie session)
	Engine        string // "portal_v4" or "old_api"
	Message       string
	Raw           string
}

// LoginManager handles Dr.COM authentication.
type LoginManager struct {
	client *http.Client
}

// NewLoginManager creates a new LoginManager.
func NewLoginManager() *LoginManager {
	return &LoginManager{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// extractACInfo parses the Portal page HTML to extract BRAS info.
func extractACInfo(html string, redirectURL string) *ACInfo {
	info := &ACInfo{
		IP:   "10.32.255.10",    // default fallback
		Name: "HJ-BRAS-ME60-01", // default fallback
	}

	// Parse from HTML
	reIP := regexp.MustCompile(`wlanacip\s*=\s*'([^']*)'`)
	reName := regexp.MustCompile(`wlanacname\s*=\s*'([^']*)'`)

	if m := reIP.FindStringSubmatch(html); len(m) >= 2 {
		info.IP = m[1]
	}
	if m := reName.FindStringSubmatch(html); len(m) >= 2 {
		info.Name = m[1]
	}

	// Parse BRAS-provided MAC, areaID, and user IP from redirect URL (more authoritative than local detection)
	// URL format: http://10.0.1.5/a79.htm?wlanuserip=10.34.93.144&wlanacname=HJ-BRAS-ME60-01&wlanacip=10.32.255.10&wlanusermac=84-9e-56-c9-11-81&areaID=ethtrunk/102:2757.0
	if redirectURL != "" {
		// BRAS user IP — the IP the BRAS actually sees for this client
		reUserIP := regexp.MustCompile(`wlanuserip=([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)`)
		if m := reUserIP.FindStringSubmatch(redirectURL); len(m) >= 2 {
			info.UserIP = m[1]
		}

		reMAC := regexp.MustCompile(`wlanusermac=([0-9a-fA-F]{2}[-:][0-9a-fA-F]{2}[-:][0-9a-fA-F]{2}[-:][0-9a-fA-F]{2}[-:][0-9a-fA-F]{2}[-:][0-9a-fA-F]{2})`)
		if m := reMAC.FindStringSubmatch(redirectURL); len(m) >= 2 {
			info.MACRaw = strings.ToUpper(m[1])                                   // preserve original format: "84-9E-56-C9-11-81"
			info.MAC = strings.ReplaceAll(strings.ReplaceAll(info.MACRaw, "-", ""), ":", "") // strip separators: "849E56C91181"
		}

		reArea := regexp.MustCompile(`areaID=([^&]+)`)
		if m := reArea.FindStringSubmatch(redirectURL); len(m) >= 2 {
			info.AreaID = m[1]
		}
	}

	return info
}

// fetchACInfo fetches the Portal page to extract AC information.
// Uses a client that does NOT follow redirects, and accesses an external URL
// (1.2.3.4) to trigger the captive portal redirect. The redirect Location header
// contains the BRAS-provided MAC, areaID, and other parameters.
func (lm *LoginManager) fetchACInfo(gateway string) *ACInfo {
	// Use a client that doesn't follow redirects so we can capture the Location header
	noRedirectClient := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // don't follow, return the redirect response
		},
	}

	body := []byte{}
	var redirectURL string

	// Try external URLs to trigger captive portal redirect.
	// The captive portal intercepts HTTP requests to external sites and redirects
	// to http://10.0.1.5/a79.htm?wlanuserip=...&wlanusermac=...&areaID=...
	// Accessing the gateway directly returns the login page WITHOUT the redirect params.
	triggerURLs := []string{
		"http://1.2.3.4/",
		"http://www.baidu.com/",
		fmt.Sprintf("http://%s/", gateway), // fallback: direct gateway access
	}

	for _, url := range triggerURLs {
		resp, err := noRedirectClient.Get(url)
		if err != nil {
			GetLogger().Debug("fetchACInfo: %s failed: %v", url, err)
			continue
		}

		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
		resp.Body.Close()

		// Check for redirect (3xx with Location header)
		loc := resp.Header.Get("Location")
		if loc != "" {
			redirectURL = loc
			body = bodyBytes
			GetLogger().Info("Captive portal redirect (from %s): %s", url, loc)
			break
		}

		// Even without redirect, save the body (may contain AC info in HTML/JS)
		if len(bodyBytes) > 0 {
			body = bodyBytes
		}

		// If we got a 2xx response, this URL works — use its body
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			break
		}
	}

	if redirectURL == "" && len(body) == 0 {
		GetLogger().Warn("Failed to fetch Portal page from any URL")
		return &ACInfo{IP: "10.32.255.10", Name: "HJ-BRAS-ME60-01"}
	}

	info := extractACInfo(string(body), redirectURL)
	GetLogger().Info("AC info: IP=%s, Name=%s, UserIP=%s, MAC=%s, MACRaw=%s, AreaID=%s", info.IP, info.Name, info.UserIP, info.MAC, info.MACRaw, info.AreaID)
	return info
}

// LoginStatus represents the parsed login result from the server.
type LoginStatus int

const (
	LoginFailed        LoginStatus = iota // result != 1
	LoginSuccess                          // result == 1
	LoginAlreadyOnline                    // client is already authenticated
)

// parseLoginResponse parses the HTTP response body and returns the login status.
func parseLoginResponse(statusCode int, body []byte) (LoginStatus, string) {
	if statusCode != 200 {
		return LoginFailed, ""
	}

	bodyStr := string(body)

	// 1. Reject HTML responses (captive portal intercept)
	if strings.HasPrefix(bodyStr, "<!DOCTYPE") || strings.HasPrefix(bodyStr, "<html") ||
		strings.HasPrefix(bodyStr, "<HTML") || strings.Contains(bodyStr, "<!DOCTYPE html") {
		GetLogger().Warn("Received HTML page instead of JSONP — captive portal interception")
		return LoginFailed, ""
	}

	// 2. Parse JSONP: callback({...})
	start := strings.Index(bodyStr, "{")
	end := strings.LastIndex(bodyStr, "}")
	if start < 0 || end <= start {
		return LoginFailed, ""
	}
	jsonPart := bodyStr[start : end+1]

	// 3. Check result + msga fields
	var response struct {
		Result interface{} `json:"result"`
		Msg    interface{} `json:"msg"`
		Msga   string      `json:"msga"`
	}
	if err := json.Unmarshal([]byte(jsonPart), &response); err != nil {
		GetLogger().Warn("Failed to parse JSONP response: %v (body: %.200s)", err, jsonPart)
		return LoginFailed, ""
	}

	// Check for "already online" indicator
	if response.Msga != "" {
		msgaLower := strings.ToLower(response.Msga)
		if strings.Contains(msgaLower, "online") {
			GetLogger().Info("Server reports client is already online: %s", response.Msga)
			return LoginAlreadyOnline, response.Msga
		}
	}
	// msg=1 often means "already online" on some Dr.COM versions
	if msgNum, ok := response.Msg.(float64); ok && msgNum == 1 {
		// Could be "already online" — check if result is 0 with msg=1
		if resultNum, ok := response.Result.(float64); ok && resultNum == 0 {
			GetLogger().Info("Server returned result=0, msg=1 — likely already online")
			return LoginAlreadyOnline, "Already online (server code msg=1)"
		}
	}

	switch v := response.Result.(type) {
	case float64:
		if v == 1 {
			return LoginSuccess, ""
		}
		GetLogger().Info("Login rejected: result=%v, msg=%v", v, response.Msg)
		return LoginFailed, fmt.Sprintf("result=%v, msg=%v", v, response.Msg)
	case string:
		if v == "1" || v == "ok" {
			return LoginSuccess, ""
		}
		GetLogger().Info("Login rejected: result=%q, msg=%v", v, response.Msg)
		return LoginFailed, fmt.Sprintf("result=%q, msg=%v", v, response.Msg)
	default:
		GetLogger().Info("Login rejected: unexpected result type %T, msg=%v", response.Result, response.Msg)
		return LoginFailed, ""
	}
}

// portalV4Login tries Portal v4.0 login (WiFi, port 801).
func (lm *LoginManager) portalV4Login(gateway, username, operator, password, ip, mac string, acInfo *ACInfo) *LoginResult {
	userAccount := fmt.Sprintf(",0,%s%s", username, OperatorToSuffix(operator))
	encodedPass := base64.StdEncoding.EncodeToString([]byte(password))

	// Prefer BRAS-provided IP over locally detected IP (critical for AC validation)
	effectiveIP := ip
	if acInfo.UserIP != "" {
		effectiveIP = acInfo.UserIP
		GetLogger().Info("Using BRAS-provided IP: %s (local: %s)", effectiveIP, ip)
	}

	// Prefer BRAS-provided MAC over locally detected MAC
	// Use MACRaw (with dashes) for Portal v4.0 — matches what BRAS provides
	effectiveMAC := mac
	if acInfo.MACRaw != "" {
		effectiveMAC = acInfo.MACRaw
		GetLogger().Info("Using BRAS-provided MAC (raw): %s (local: %s)", effectiveMAC, mac)
	} else if acInfo.MAC != "" {
		effectiveMAC = acInfo.MAC
		GetLogger().Info("Using BRAS-provided MAC: %s (local: %s)", effectiveMAC, mac)
	}

	url := fmt.Sprintf(
		"http://%s:801/eportal/portal/login?callback=dr1004&login_method=1"+
			"&user_account=%s"+
			"&user_password=%s"+
			"&wlan_user_ip=%s"+
			"&wlan_user_mac=%s"+
			"&wlan_ac_ip=%s"+
			"&wlan_ac_name=%s",
		gateway, userAccount, encodedPass, effectiveIP, effectiveMAC, acInfo.IP, acInfo.Name,
	)

	// Append areaID if available (required by some Dr.COM deployments for WiFi)
	if acInfo.AreaID != "" {
		url += "&wlan_area_id=" + acInfo.AreaID
		GetLogger().Info("Appending areaID: %s", acInfo.AreaID)
	}

	url += "&terminal_type=1&jsVersion=4.2&v=8746"

	GetLogger().Info("Portal v4.0 login attempt: %s", gateway)
	GetLogger().Info("Portal URL: %s", url)

	resp, err := lm.client.Get(url)
	if err != nil {
		return &LoginResult{Success: false, Engine: "portal_v4", Message: fmt.Sprintf("Network error: %v", err)}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	status, detail := parseLoginResponse(resp.StatusCode, body)
	switch status {
	case LoginSuccess:
		GetLogger().Info("Portal v4.0 login SUCCESS")
		return &LoginResult{Success: true, Engine: "portal_v4", Message: "Login successful (Portal v4.0)", Raw: string(body)}
	case LoginAlreadyOnline:
		GetLogger().Info("Portal v4.0: already online — %s", detail)
		return &LoginResult{Success: true, AlreadyOnline: true, Engine: "portal_v4", Message: "Already online", Raw: string(body)}
	}

	bodyPreview := string(body)
	if len(bodyPreview) > 300 {
		bodyPreview = bodyPreview[:300]
	}
	GetLogger().Warn("Portal v4.0 login failed, body: %s", bodyPreview)
	return &LoginResult{
		Success: false,
		Engine:  "portal_v4",
		Message: fmt.Sprintf("Portal v4.0 rejected: %s", detail),
		Raw:     string(body),
	}
}

// oldAPILogin tries the legacy Dr.COM login API (wired, port 80).
func (lm *LoginManager) oldAPILogin(gateway, username, operator, password, ip, v6ip, mac string, acInfo *ACInfo) *LoginResult {
	account := fmt.Sprintf("%s%s", username, OperatorToSuffix(operator))

	// Campus network uses R6=0 + terminal_type=1; carriers use R6=1 + terminal_type=2
	r6 := "1"
	termType := "2"
	if operator == "campus" {
		r6 = "0"
		termType = "1"
	}

	// Prefer BRAS-provided IP over locally detected IP
	effectiveIP := ip
	if acInfo.UserIP != "" {
		effectiveIP = acInfo.UserIP
	}

	// Prefer BRAS-provided MAC over locally detected MAC
	effectiveMAC := mac
	if acInfo.MAC != "" {
		effectiveMAC = acInfo.MAC
	}

	url := fmt.Sprintf(
		"http://%s:80/drcom/login?callback=dr1004"+
			"&DDDDD=%s"+
			"&upass=%s"+
			"&0MKKey=123456"+
			"&R1=0&R2=&R3=0&R6=%s&para=00"+
			"&v4ip=%s"+
			"&terminal_type=%s&lang=zh-cn&jsVersion=4.2&v=608",
		gateway, account, password, r6, effectiveIP, termType,
	)

	// Append v6ip for campus network
	if operator == "campus" && v6ip != "" {
		url += "&v6ip=" + v6ip
	}

	// Append MAC for old API
	if effectiveMAC != "" {
		url += "&v4mac=" + effectiveMAC
	}

	GetLogger().Info("Old API login attempt: %s", gateway)
	GetLogger().Info("Old API URL: %s", url)

	resp, err := lm.client.Get(url)
	if err != nil {
		return &LoginResult{Success: false, Engine: "old_api", Message: fmt.Sprintf("Network error: %v", err)}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	status, detail := parseLoginResponse(resp.StatusCode, body)
	switch status {
	case LoginSuccess:
		GetLogger().Info("Old API login SUCCESS")
		return &LoginResult{Success: true, Engine: "old_api", Message: "Login successful (Old API)", Raw: string(body)}
	case LoginAlreadyOnline:
		GetLogger().Info("Old API: already online — %s", detail)
		return &LoginResult{Success: true, AlreadyOnline: true, Engine: "old_api", Message: "Already online", Raw: string(body)}
	}

	bodyPreview := string(body)
	if len(bodyPreview) > 300 {
		bodyPreview = bodyPreview[:300]
	}
	GetLogger().Warn("Old API login failed, body: %s", bodyPreview)
	return &LoginResult{
		Success: false,
		Engine:  "old_api",
		Message: fmt.Sprintf("Old API rejected: %s", detail),
		Raw:     string(body),
	}
}

// Login performs the dual-engine login sequence.
// 1. Fetch AC info from Portal page
// 2. Try Portal v4.0 (WiFi, port 801)
// 3. Fall back to old API (wired, port 80)
func (lm *LoginManager) Login(gateway, username, operator, password, localIP, localMAC, v6ip string) *LoginResult {
	if localIP == "" {
		return &LoginResult{Success: false, Message: "No active network interface found"}
	}

	// Step 1: Fetch AC info from Portal page (for Portal v4.0 parameters)
	acInfo := lm.fetchACInfo(gateway)

	// Step 2: Try Portal v4.0 first (required for WiFi)
	GetLogger().Info("Attempting Portal v4.0 login (gateway=%s, user=%s, operator=%s)", gateway, username, operator)
	result := lm.portalV4Login(gateway, username, operator, password, localIP, localMAC, acInfo)
	if result.Success {
		return result
	}
	GetLogger().Warn("Portal v4.0 failed: [%s] %s", result.Engine, result.Message)

	// Step 3: Fall back to old API (for wired connections)
	GetLogger().Info("Falling back to old API login")
	result = lm.oldAPILogin(gateway, username, operator, password, localIP, v6ip, localMAC, acInfo)
	if result.Success {
		return result
	}
	GetLogger().Warn("Old API failed: [%s] %s", result.Engine, result.Message)

	// Both failed — log the old API raw response for debugging
	GetLogger().Error("All login methods exhausted. Check gateway=%s, username=%s, operator=%s", gateway, username, operator)
	return &LoginResult{
		Success: false,
		Engine:  "dual",
		Message: "All login methods failed. Check credentials and network.",
	}
}

// =============================================================================
// Dr.COM Logout — Clear stale sessions
// =============================================================================

// portalV4Logout sends a logout request to the Portal v4.0 endpoint.
func (lm *LoginManager) portalV4Logout(gateway, username, operator, ip, mac string, acInfo *ACInfo) bool {
	userAccount := fmt.Sprintf(",0,%s%s", username, OperatorToSuffix(operator))

	effectiveIP := ip
	if acInfo.UserIP != "" {
		effectiveIP = acInfo.UserIP
	}

	effectiveMAC := mac
	if acInfo.MACRaw != "" {
		effectiveMAC = acInfo.MACRaw
	} else if acInfo.MAC != "" {
		effectiveMAC = acInfo.MAC
	}

	url := fmt.Sprintf(
		"http://%s:801/eportal/portal/logout?callback=dr1004&login_method=1"+
			"&user_account=%s"+
			"&wlan_user_ip=%s"+
			"&wlan_user_mac=%s"+
			"&wlan_ac_ip=%s"+
			"&wlan_ac_name=%s",
		gateway, userAccount, effectiveIP, effectiveMAC, acInfo.IP, acInfo.Name,
	)

	// Append areaID if available
	if acInfo.AreaID != "" {
		url += "&wlan_area_id=" + acInfo.AreaID
	}

	url += "&terminal_type=1&jsVersion=4.2&v=8746"

	GetLogger().Info("Portal v4.0 logout: %s", gateway)

	resp, err := lm.client.Get(url)
	if err != nil {
		GetLogger().Warn("Portal v4.0 logout network error: %v", err)
		return false
	}
	defer resp.Body.Close()

	GetLogger().Info("Portal v4.0 logout response: HTTP %d", resp.StatusCode)
	return resp.StatusCode == 200
}

// oldAPILogout sends a logout request to the legacy Dr.COM API endpoint.
func (lm *LoginManager) oldAPILogout(gateway, username, operator, ip, mac string, acInfo *ACInfo) bool {
	account := fmt.Sprintf("%s%s", username, OperatorToSuffix(operator))

	effectiveIP := ip
	if acInfo.UserIP != "" {
		effectiveIP = acInfo.UserIP
	}

	effectiveMAC := mac
	if acInfo.MAC != "" {
		effectiveMAC = acInfo.MAC
	}

	url := fmt.Sprintf(
		"http://%s:80/drcom/logout?callback=dr1004"+
			"&DDDDD=%s"+
			"&v4ip=%s"+
			"&v4mac=%s"+
			"&lang=zh-cn",
		gateway, account, effectiveIP, effectiveMAC,
	)

	GetLogger().Info("Old API logout: %s", gateway)

	resp, err := lm.client.Get(url)
	if err != nil {
		GetLogger().Warn("Old API logout network error: %v", err)
		return false
	}
	defer resp.Body.Close()

	GetLogger().Info("Old API logout response: HTTP %d", resp.StatusCode)
	return resp.StatusCode == 200
}

// Logout attempts to clear any existing session on the BRAS.
// It tries Portal v4.0 first, then falls back to old API.
func (lm *LoginManager) Logout(gateway, username, operator, ip, mac string) bool {
	if ip == "" {
		GetLogger().Warn("Logout skipped: no IP address")
		return false
	}

	acInfo := lm.fetchACInfo(gateway)

	GetLogger().Info("Logging out (gateway=%s, user=%s)", gateway, username)

	// Try Portal v4.0 first
	if lm.portalV4Logout(gateway, username, operator, ip, mac, acInfo) {
		GetLogger().Info("Logout successful via Portal v4.0")
		return true
	}

	// Fall back to old API
	if lm.oldAPILogout(gateway, username, operator, ip, mac, acInfo) {
		GetLogger().Info("Logout successful via Old API")
		return true
	}

	GetLogger().Warn("Logout: both engines returned non-200 (may still have succeeded)")
	return false
}
