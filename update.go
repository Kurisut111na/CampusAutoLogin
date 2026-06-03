package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/lxn/walk"
)

// =============================================================================
// Version Check — zero-server update mechanism via GitHub raw JSON
// =============================================================================
//
// A static version.json in a public GitHub repo serves as the update manifest.
// No server-side code, no deployment. Just edit a JSON file and git push.
//
// JSON format:
//
//	{
//	  "latest":       "1.3.0",
//	  "min":          "1.0.0",
//	  "download":     "https://your-lanzou-link",
//	  "release_note": "修复断线重连bug"
//	}

// AppVersion is the current version — bump this every release.
const AppVersion = "1.0.0"

// VersionCheckURL is the raw GitHub URL for the version manifest.
// TODO: replace with your actual repo URL.
const VersionCheckURL = "https://raw.githubusercontent.com/Kurisut111na/CampusAutoLogin/main/version.json"

type versionManifest struct {
	Latest      string `json:"latest"`
	Min         string `json:"min"`
	Download    string `json:"download"`
	ReleaseNote string `json:"release_note"`
}

// cached manifest from startup check, used by ShowUpdateNotification later
var cachedManifest *versionManifest

// =============================================================================
// Startup check (synchronous, with timeout — called BEFORE window creation)
// =============================================================================

// CheckVersionSync fetches the remote manifest and enforces minimum version.
// Returns true if the app should exit (force-update required).
// Runs synchronously before the main window is created, with a 5-second timeout.
func CheckVersionSync() bool {
	manifest, err := fetchManifest()
	if err != nil {
		// Network down → silently skip, don't block the user
		GetLogger().Debug("Version check skipped (network): %v", err)
		return false
	}

	current := parseVersion(AppVersion)
	min := parseVersion(manifest.Min)
	latest := parseVersion(manifest.Latest)

	GetLogger().Info("Version check: current=%s remote latest=%s min=%s",
		AppVersion, manifest.Latest, manifest.Min)

	// Force update — version below minimum
	if versionLess(current, min) {
		showForceUpdate(manifest)
		return true // caller should os.Exit
	}

	// Update available but not forced — cache for later notification
	if versionLess(current, latest) {
		cachedManifest = manifest
	}

	return false
}

// =============================================================================
// Deferred notification (called AFTER window is ready)
// =============================================================================

// ShowUpdateNotification shows an optional update dialog if a new version
// is available. Call this after the main window has been created.
func ShowUpdateNotification(owner walk.Form) {
	if cachedManifest == nil {
		return
	}
	showUpdateAvailable(owner, cachedManifest)
}

// =============================================================================
// Network fetch
// =============================================================================

func fetchManifest() (*versionManifest, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(VersionCheckURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var m versionManifest
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}

// =============================================================================
// Version comparison
// =============================================================================

func parseVersion(v string) []int {
	parts := strings.Split(strings.TrimSpace(v), ".")
	result := make([]int, len(parts))
	for i, p := range parts {
		n, _ := strconv.Atoi(p)
		result[i] = n
	}
	return result
}

func versionLess(a, b []int) bool {
	n := len(a)
	if len(b) > n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		va := 0
		if i < len(a) {
			va = a[i]
		}
		vb := 0
		if i < len(b) {
			vb = b[i]
		}
		if va != vb {
			return va < vb
		}
	}
	return false
}

// =============================================================================
// Dialogs
// =============================================================================

func showForceUpdate(m *versionManifest) {
	msg := fmt.Sprintf(
		"您正在使用的版本 v%s 已不再受支持。\n\n"+
			"最新版本：v%s\n\n"+
			"%s\n\n"+
			"请下载最新版本后继续使用。",
		AppVersion, m.Latest, m.ReleaseNote,
	)
	walk.MsgBox(nil, "版本已过期 — 请更新", msg, walk.MsgBoxIconWarning)
}

func showUpdateAvailable(owner walk.Form, m *versionManifest) {
	msg := fmt.Sprintf(
		"发现新版本 v%s（当前 v%s）\n\n"+
			"%s\n\n"+
			"是否前往下载？",
		m.Latest, AppVersion, m.ReleaseNote,
	)
	result := walk.MsgBox(owner, "发现新版本", msg,
		walk.MsgBoxIconInformation|walk.MsgBoxOKCancel)
	if result == walk.DlgCmdOK {
		openBrowser(m.Download)
	}
}

func openBrowser(url string) {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		GetLogger().Warn("Failed to open browser: %v", err)
	}
}
