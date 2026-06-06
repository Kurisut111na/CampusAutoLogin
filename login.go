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
	IP   string
	Name string
}

// LoginResult holds the result of a login attempt.
type LoginResult struct {
	Success bool
	Engine  string // "portal_v4" or "old_api"
	Message string
	Raw     string
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
func extractACInfo(html string) *ACInfo {
	info := &ACInfo{
		IP:   "10.32.255.10",    // default fallback
		Name: "HJ-BRAS-ME60-01", // default fallback
	}

	reIP := regexp.MustCompile(`wlanacip\s*=\s*'([^']*)'`)
	reName := regexp.MustCompile(`wlanacname\s*=\s*'([^']*)'`)

	if m := reIP.FindStringSubmatch(html); len(m) >= 2 {
		info.IP = m[1]
	}
	if m := reName.FindStringSubmatch(html); len(m) >= 2 {
		info.Name = m[1]
	}

	return info
}

// fetchACInfo fetches the Portal page to extract AC information.
func (lm *LoginManager) fetchACInfo(gateway string) *ACInfo {
	url := fmt.Sprintf("http://%s/", gateway)
	resp, err := lm.client.Get(url)
	if err != nil {
		GetLogger().Warn("Failed to fetch Portal page from %s: %v", gateway, err)
		return &ACInfo{IP: "10.32.255.10", Name: "HJ-BRAS-ME60-01"}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	info := extractACInfo(string(body))
	GetLogger().Debug("AC info: IP=%s, Name=%s", info.IP, info.Name)
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

	url := fmt.Sprintf(
		"http://%s:801/eportal/portal/login?callback=dr1004&login_method=1"+
			"&user_account=%s"+
			"&user_password=%s"+
			"&wlan_user_ip=%s"+
			"&wlan_user_mac=%s"+
			"&wlan_ac_ip=%s"+
			"&wlan_ac_name=%s"+
			"&terminal_type=1&jsVersion=4.2&v=8746",
		gateway, userAccount, encodedPass, ip, mac, acInfo.IP, acInfo.Name,
	)

	GetLogger().Info("Portal v4.0 login attempt: %s", gateway)
	GetLogger().Debug("Portal URL: %s", url)

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
		return &LoginResult{Success: true, Engine: "portal_v4", Message: "Already online", Raw: string(body)}
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
func (lm *LoginManager) oldAPILogin(gateway, username, operator, password, ip, v6ip string) *LoginResult {
	account := fmt.Sprintf("%s%s", username, OperatorToSuffix(operator))

	// Campus network uses R6=0 + terminal_type=1; carriers use R6=1 + terminal_type=2
	r6 := "1"
	termType := "2"
	if operator == "campus" {
		r6 = "0"
		termType = "1"
	}

	url := fmt.Sprintf(
		"http://%s:80/drcom/login?callback=dr1004"+
			"&DDDDD=%s"+
			"&upass=%s"+
			"&0MKKey=123456"+
			"&R1=0&R2=&R3=0&R6=%s&para=00"+
			"&v4ip=%s"+
			"&terminal_type=%s&lang=zh-cn&jsVersion=4.2&v=608",
		gateway, account, password, r6, ip, termType,
	)

	// Append v6ip for campus network
	if operator == "campus" && v6ip != "" {
		url += "&v6ip=" + v6ip
	}

	GetLogger().Info("Old API login attempt: %s", gateway)
	GetLogger().Debug("Old API URL: %s", url)

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
		return &LoginResult{Success: true, Engine: "old_api", Message: "Already online", Raw: string(body)}
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
	result = lm.oldAPILogin(gateway, username, operator, password, localIP, v6ip)
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
