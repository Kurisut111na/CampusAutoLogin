package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// =============================================================================
// Windows Registry Auto-Start (via reg.exe — the reliable way)
// =============================================================================
//
// Using reg.exe instead of raw Win32 syscalls avoids registry API pitfalls:
// access masks, 32/64-bit redirection, handle lifetime, and encoding issues.
// reg.exe is the officially supported CLI and has been rock-solid since Win2K.

const regRunKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
const autoStartValueName = "CampusAutoLogin"

// RegisterAutoStart adds the program to Windows startup.
// Pass --silent so auto-start runs minimized.
func RegisterAutoStart() error {
	// Get current exe path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get exe path: %w", err)
	}
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("abs exe path: %w", err)
	}

	// Quote the path and add --silent flag
	value := fmt.Sprintf(`"%s" --silent`, exePath)

	// reg add HKCU\Software\Microsoft\Windows\CurrentVersion\Run /v CampusAutoLogin /t REG_SZ /d "value" /f
	cmd := exec.Command("reg", "add",
		regRunKey,
		"/v", autoStartValueName,
		"/t", "REG_SZ",
		"/d", value,
		"/f",
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("reg add failed: %w\n%s", err, string(output))
	}

	GetLogger().Info("Auto-start registered: %s", value)
	return nil
}

// UnregisterAutoStart removes the program from Windows startup.
func UnregisterAutoStart() error {
	// reg delete HKCU\Software\Microsoft\Windows\CurrentVersion\Run /v CampusAutoLogin /f
	cmd := exec.Command("reg", "delete",
		regRunKey,
		"/v", autoStartValueName,
		"/f",
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	output, err := cmd.CombinedOutput()
	if err != nil {
		// The value might not exist — that's fine
		GetLogger().Info("Auto-start unregister (value may not exist): %s", string(output))
		return nil
	}

	GetLogger().Info("Auto-start unregistered")
	return nil
}

// IsSilentStart checks if the app was launched with --silent flag (from auto-start).
func IsSilentStart() bool {
	for _, arg := range os.Args[1:] {
		if arg == "--silent" {
			return true
		}
	}
	return false
}
