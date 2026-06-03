package main

import (
	"os"
	"strings"
	"time"

	"github.com/lxn/walk"
)

// =============================================================================
// Shared Constants
// =============================================================================

const (
	connTimeout = 3 * time.Second // TCP dial timeout for gateway probing
)

// =============================================================================
// Entry Point
// =============================================================================

func main() {
	_ = walk.App()

	if err := InitLogger(LogInfo); err != nil {
		println("Warning: failed to init logger:", err.Error())
	}
	GetLogger().Info("CampusAutoLogin v%s starting", AppVersion)

	// Check for updates before creating the window.
	// If force-update is required, this shows a dialog and returns true.
	if CheckVersionSync() {
		GetLogger().Close()
		os.Exit(0)
	}

	configMgr, err := NewConfigManager()
	if err != nil {
		GetLogger().Error("Failed to create config manager: %v", err)
		os.Exit(1)
	}

	loginMgr := NewLoginManager()
	heartbeat := NewHeartbeat()

	mainWin, err := NewMainWindow(configMgr, loginMgr, heartbeat, nil)
	if err != nil {
		GetLogger().Error("Failed to create main window: %v", err)
		os.Exit(1)
	}
	defer mainWin.Dispose()

	tray, err := NewTrayIcon(mainWin)
	if err != nil {
		GetLogger().Error("Failed to create tray icon: %v", err)
		os.Exit(1)
	}
	defer tray.Dispose()
	tray.SetVisible(true)

	mainWin.tray = tray
	mainWin.setupCallbacks()

	// Show optional update notification (non-blocking — user can ignore)
	ShowUpdateNotification(mainWin)

	// Only minimize when auto-started via registry (--silent flag),
	// not when user double-clicks manually.
	if IsSilentStart() && mainWin.ShouldStartMinimized() {
		GetLogger().Info("Starting minimized to tray (auto-start)")
		mainWin.Hide()
	} else {
		mainWin.Show()
	}

	mainWin.TriggerAutoLogin()

	GetLogger().Info("Entering message loop")
	mainWin.Run()

	GetLogger().Info("Application exiting")
	mainWin.saveConfigFromUI()
	heartbeat.Stop()
	GetLogger().Close()
}

// =============================================================================
// Utility functions (used across files)
// =============================================================================

// splitLines splits a string by newlines (handles \r\n and \n).
func splitLines(s string) []string {
	return strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
}

// trimSpace trims whitespace from a string.
func trimSpace(s string) string {
	return strings.TrimSpace(s)
}
