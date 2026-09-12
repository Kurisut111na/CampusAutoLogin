package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lxn/walk"
)

//go:embed assets/icon_green.ico assets/icon_blue.ico assets/icon_red.ico
var iconAssets embed.FS

// =============================================================================
// System Tray Icon
// =============================================================================

// TrayIcon manages the system tray notification area icon.
type TrayIcon struct {
	ni            *walk.NotifyIcon
	iconConnected *walk.Icon
	iconLoggedIn  *walk.Icon
	iconLost      *walk.Icon

	loggedIn  bool
	connected bool

	// Callbacks
	onShowWindow func()
	onReconnect  func()
	onOpenLogDir func()
	onQuit       func()
}

// NewTrayIcon creates a new system tray icon.
// The form parameter is the main window (required by walk.NotifyIcon).
func NewTrayIcon(form walk.Form) (*TrayIcon, error) {
	ti := &TrayIcon{}

	// Generate colored icons from embedded assets
	var err error
	ti.iconConnected, err = generateColorIcon("green")
	if err != nil {
		return nil, fmt.Errorf("generate green icon: %w", err)
	}
	ti.iconLoggedIn, err = generateColorIcon("blue")
	if err != nil {
		return nil, fmt.Errorf("generate blue icon: %w", err)
	}
	ti.iconLost, err = generateColorIcon("red")
	if err != nil {
		return nil, fmt.Errorf("generate red icon: %w", err)
	}

	// Create the notify icon
	ni, err := walk.NewNotifyIcon(form)
	if err != nil {
		return nil, fmt.Errorf("NewNotifyIcon: %w", err)
	}
	ti.ni = ni

	ni.SetToolTip("Campus Auto Login")
	ni.SetIcon(ti.iconLoggedIn) // start with blue (logged in state default)

	// Build the context menu
	ti.buildMenu()

	// Double-click: show window
	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			if ti.onShowWindow != nil {
				ti.onShowWindow()
			}
		}
	})

	return ti, nil
}

// Dispose cleans up resources.
func (ti *TrayIcon) Dispose() {
	if ti.ni != nil {
		ti.ni.Dispose()
	}
}

// SetVisible shows or hides the tray icon.
func (ti *TrayIcon) SetVisible(visible bool) {
	if ti.ni != nil {
		ti.ni.SetVisible(visible)
	}
}

// SetLoggedIn updates the login state (affects icon color).
func (ti *TrayIcon) SetLoggedIn(loggedIn bool) {
	ti.loggedIn = loggedIn
	ti.updateIcon()
}

// SetConnectionStatus updates the connection state (affects icon color).
func (ti *TrayIcon) SetConnectionStatus(connected bool) {
	ti.connected = connected
	ti.updateIcon()
}

func (ti *TrayIcon) updateIcon() {
	if ti.ni == nil {
		return
	}
	var icon *walk.Icon
	if !ti.loggedIn {
		icon = ti.iconLost
	} else if !ti.connected {
		icon = ti.iconLost
	} else {
		icon = ti.iconConnected
	}
	ti.ni.SetIcon(icon)

	if ti.connected && ti.loggedIn {
		ti.ni.SetToolTip("Campus Auto Login — Connected")
	} else {
		ti.ni.SetToolTip("Campus Auto Login — Disconnected")
	}
}

// ShowBalloon displays a notification balloon.
func (ti *TrayIcon) ShowBalloon(title, message string) {
	if ti.ni != nil {
		ti.ni.ShowInfo(title, message)
	}
}

// =============================================================================
// Callback setters
// =============================================================================

func (ti *TrayIcon) OnShowWindow(fn func()) { ti.onShowWindow = fn }
func (ti *TrayIcon) OnReconnect(fn func())  { ti.onReconnect = fn }
func (ti *TrayIcon) OnOpenLogDir(fn func()) { ti.onOpenLogDir = fn }
func (ti *TrayIcon) OnQuit(fn func())       { ti.onQuit = fn }

// =============================================================================
// Context Menu (built directly on the NotifyIcon)
// =============================================================================

func (ti *TrayIcon) buildMenu() {
	// Show/Hide
	showAction := walk.NewAction()
	showAction.SetText("显示/隐藏窗口")
	showAction.Triggered().Attach(func() {
		if ti.onShowWindow != nil {
			ti.onShowWindow()
		}
	})
	ti.ni.ContextMenu().Actions().Add(showAction)

	// Reconnect
	reconnectAction := walk.NewAction()
	reconnectAction.SetText("重新连接")
	reconnectAction.Triggered().Attach(func() {
		if ti.onReconnect != nil {
			ti.onReconnect()
		}
	})
	ti.ni.ContextMenu().Actions().Add(reconnectAction)

	// Open Log Dir
	openLogAction := walk.NewAction()
	openLogAction.SetText("打开日志目录")
	openLogAction.Triggered().Attach(func() {
		if ti.onOpenLogDir != nil {
			ti.onOpenLogDir()
		}
	})
	ti.ni.ContextMenu().Actions().Add(openLogAction)

	// Separator
	ti.ni.ContextMenu().Actions().Add(walk.NewSeparatorAction())

	// Quit
	quitAction := walk.NewAction()
	quitAction.SetText("退出")
	quitAction.Triggered().Attach(func() {
		if ti.onQuit != nil {
			ti.onQuit()
		}
	})
	ti.ni.ContextMenu().Actions().Add(quitAction)
}

// =============================================================================
// =============================================================================
// Icon Loading — pre-rendered multi-resolution .ico assets (assets/*.ico)
// =============================================================================
//
// 三色图标由 assets/ 内的 .ico 提供（16/24/32/48/64 五尺寸、4x 超采样抗
// 锯齿，由原运行时编码器一次性固化生成）。walk 的 LoadImage 只支持从文件
// 加载多尺寸 Icon，无法走内存，故首次使用时将 embed 字节落盘到 %TEMP%。

func generateColorIcon(name string) (*walk.Icon, error) {
	icoFile := filepath.Join(os.TempDir(), "campus_ico_v3_"+name+".ico")
	if _, err := os.Stat(icoFile); os.IsNotExist(err) {
		data, err := iconAssets.ReadFile("assets/icon_" + name + ".ico")
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(icoFile, data, 0644); err != nil {
			return nil, err
		}
	}
	return walk.NewIconFromFile(icoFile)
}
