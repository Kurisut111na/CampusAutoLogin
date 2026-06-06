package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os/exec"
	"time"

	"github.com/lxn/walk"
)

//go:embed 3145ba5c607a9be9f94794db7160bd85.png
var qrCodePNG []byte

// =============================================================================
// Main Window — Campus Auto Login
// =============================================================================

// MainWindow is the application's primary window.
type MainWindow struct {
	*walk.MainWindow

	// Managers
	configMgr *ConfigManager
	loginMgr  *LoginManager
	heartbeat *Heartbeat
	tray      *TrayIcon

	// Config (live copy)
	cfg *AppConfig

	// Session password (available even if not persisted)
	sessionPassword string

	// --- Account Section ---
	usernameEdit    *walk.LineEdit
	passwordEdit    *walk.LineEdit
	showPasswordBtn *walk.PushButton
	operatorCombo   *walk.ComboBox

	// --- Options ---
	autoLoginCheck   *walk.CheckBox
	rememberPwdCheck *walk.CheckBox
	heartbeatCheck   *walk.CheckBox

	// --- Status ---
	statusLabel     *walk.LineEdit
	connectionLabel *walk.LineEdit
	ipLabel         *walk.LineEdit
	gatewayLabel    *walk.LineEdit
	lastLoginLabel  *walk.LineEdit

	// --- Buttons ---
	loginBtn     *walk.PushButton
	reconnectBtn *walk.PushButton

	// --- Log Panel ---
	logView       *LogView
	logLevelFilter *walk.ComboBox

	// --- State ---
	passwordVisible     bool
	reconnectOnCooldown bool
	loading             bool

	// --- Timers ---
	reconnectCooldown *time.Timer
	saveTimer         *time.Timer
}

// NewMainWindow creates and initializes the main window.
func NewMainWindow(configMgr *ConfigManager, loginMgr *LoginManager,
	heartbeat *Heartbeat, tray *TrayIcon) (*MainWindow, error) {

	mw := &MainWindow{
		configMgr: configMgr,
		loginMgr:  loginMgr,
		heartbeat: heartbeat,
		tray:      tray,
	}

	// Load config
	mw.cfg = configMgr.LoadConfig()
	if mw.cfg.RememberPassword && mw.cfg.Password != "" {
		mw.sessionPassword = mw.cfg.Password
	}

	// Create the main window
	mainWin, err := walk.NewMainWindow()
	if err != nil {
		return nil, err
	}
	mw.MainWindow = mainWin

	mw.SetTitle("Campus Auto Login - 校园网自动登录")
	mw.SetMinMaxSize(walk.Size{Width: 480, Height: 520}, walk.Size{Width: 800, Height: 900})
	mw.SetSize(walk.Size{Width: 520, Height: 620})
	mw.SetLayout(walk.NewVBoxLayout())

	if err := mw.buildUI(); err != nil {
		return nil, err
	}

	mw.loadConfigToUI()
	// NOTE: setupCallbacks() is called from main() AFTER tray is linked

	return mw, nil
}

// =============================================================================
// UI Construction
// =============================================================================

func (mw *MainWindow) buildUI() error {
	// --- Account Section ---
	accountGroup, err := walk.NewGroupBox(mw)
	if err != nil {
		return err
	}
	accountGroup.SetTitle("账号 Account")
	accGrid := walk.NewGridLayout()
	accountGroup.SetLayout(accGrid)

	// Row 0: Username
	userLabel, err := walk.NewLabel(accountGroup)
	if err != nil {
		return err
	}
	userLabel.SetText("用户名 (学号):")
	accGrid.SetRange(userLabel, walk.Rectangle{X: 0, Y: 0, Width: 1, Height: 1})

	mw.usernameEdit, err = walk.NewLineEdit(accountGroup)
	if err != nil {
		return err
	}
	mw.usernameEdit.SetMaxLength(30)
	mw.usernameEdit.SetMinMaxSize(walk.Size{Width: 200, Height: 24}, walk.Size{Width: 400, Height: 24})
	accGrid.SetRange(mw.usernameEdit, walk.Rectangle{X: 1, Y: 0, Width: 1, Height: 1})

	// Row 1: Password
	passLabel, err := walk.NewLabel(accountGroup)
	if err != nil {
		return err
	}
	passLabel.SetText("密码:")
	accGrid.SetRange(passLabel, walk.Rectangle{X: 0, Y: 1, Width: 1, Height: 1})

	passRow, err := walk.NewComposite(accountGroup)
	if err != nil {
		return err
	}
	passRow.SetLayout(walk.NewHBoxLayout())

	mw.passwordEdit, err = walk.NewLineEdit(passRow)
	if err != nil {
		return err
	}
	mw.passwordEdit.SetPasswordMode(true)
	mw.passwordEdit.SetMaxLength(64)
	mw.passwordEdit.SetMinMaxSize(walk.Size{Width: 170, Height: 24}, walk.Size{Width: 370, Height: 24})

	mw.showPasswordBtn, err = walk.NewPushButton(passRow)
	if err != nil {
		return err
	}
	mw.showPasswordBtn.SetText("👁")
	mw.showPasswordBtn.SetMinMaxSize(walk.Size{Width: 30, Height: 24}, walk.Size{Width: 30, Height: 24})
	accGrid.SetRange(passRow, walk.Rectangle{X: 1, Y: 1, Width: 1, Height: 1})

	// Row 2: Operator
	opLabel, err := walk.NewLabel(accountGroup)
	if err != nil {
		return err
	}
	opLabel.SetText("运营商:")
	accGrid.SetRange(opLabel, walk.Rectangle{X: 0, Y: 2, Width: 1, Height: 1})

	mw.operatorCombo, err = walk.NewComboBox(accountGroup)
	if err != nil {
		return err
	}
	mw.operatorCombo.SetModel([]string{"中国移动", "中国联通", "中国电信", "校园网"})
	mw.operatorCombo.SetCurrentIndex(0)
	mw.operatorCombo.SetMinMaxSize(walk.Size{Width: 140, Height: 24}, walk.Size{Width: 200, Height: 24})
	accGrid.SetRange(mw.operatorCombo, walk.Rectangle{X: 1, Y: 2, Width: 1, Height: 1})

	// Set column stretch: label column fixed, input column stretches
	accGrid.SetColumnStretchFactor(0, 0)
	accGrid.SetColumnStretchFactor(1, 1)

	// --- Options Section ---
	optGroup, err := walk.NewGroupBox(mw)
	if err != nil {
		return err
	}
	optGroup.SetTitle("选项 Options")
	optGroup.SetLayout(walk.NewVBoxLayout())

	mw.autoLoginCheck, err = walk.NewCheckBox(optGroup)
	if err != nil {
		return err
	}
	mw.autoLoginCheck.SetText("启动时自动登录 (Auto-login on startup)")

	mw.rememberPwdCheck, err = walk.NewCheckBox(optGroup)
	if err != nil {
		return err
	}
	mw.rememberPwdCheck.SetText("记住密码 (Remember password)")

	// Security note — clickable LinkLabel (blue underlined, universally recognized)
	secLink, err := walk.NewLinkLabel(optGroup)
	if err != nil {
		return err
	}
	secLink.SetText(`  <a>ⓘ 密码安全说明</a> — 您的密码如何被保护？`)
	secLink.LinkActivated().Attach(func(link *walk.LinkLabelLink) {
		walk.MsgBox(mw, "密码安全说明",
			"您的账号密码使用 Windows DPAPI (CryptProtectData) 加密存储。\n\n"+
				"🔐 加密特点：\n"+
				"· 密文绑定当前 Windows 账户 + 当前设备\n"+
				"· 换电脑、换用户均无法解密\n"+
				"· 开发者无法获取您的明文密码\n"+
				"· 不依赖外部密钥文件\n\n"+
				"📂 配置文件位置：\n"+
				"%LOCALAPPDATA%\\CampusAutoLogin\\config.json\n"+
				"（密码字段为 Base64+DPAPI 密文）\n\n"+
				"本软件为开源项目，所有代码可审查，确认无后门。",
			walk.MsgBoxIconInformation)
	})

	// Heartbeat row: checkbox + info link
	hbRow, err := walk.NewComposite(optGroup)
	if err != nil {
		return err
	}
	hbRow.SetLayout(walk.NewHBoxLayout())

	mw.heartbeatCheck, err = walk.NewCheckBox(hbRow)
	if err != nil {
		return err
	}
	mw.heartbeatCheck.SetText("启用心跳检测 (Enable heartbeat)")

	hbInfo, err := walk.NewLinkLabel(hbRow)
	if err != nil {
		return err
	}
	hbInfo.SetText(`<a>ⓘ 什么是心跳检测？</a>`)
	hbInfo.LinkActivated().Attach(func(link *walk.LinkLabelLink) {
		walk.MsgBox(mw, "心跳检测说明",
			"心跳检测 (Heartbeat) 通过定期发送极小的 HTTP HEAD 请求来监测校园网连接状态。\n\n"+
				"📋 工作方式：\n"+
				"· 每隔 N 秒向百度/Bing 发送一个 HEAD 请求（仅获取响应头，不下载网页）\n"+
				"· 如果连续 2 次失败，判定为断线，自动发起重连\n"+
				"· 重连采用指数退避策略（1s→2s→4s→8s→16s，最多 5 次）\n\n"+
				"⚡ 性能影响：\n"+
				"· 每次请求体积 < 1KB，45 秒间隔下全天流量约 2MB\n"+
				"· CPU 占用可忽略不计\n"+
				"· 不会影响正常上网速度\n\n"+
				"🔧 可在「高级设置」中调整检测间隔（15-300 秒）和目标网址。",
			walk.MsgBoxIconInformation)
	})

	// --- Status Section ---
	statusGroup, err := walk.NewGroupBox(mw)
	if err != nil {
		return err
	}
	statusGroup.SetTitle("状态 Status")
	statGrid := walk.NewGridLayout()
	statusGroup.SetLayout(statGrid)

	mw.statusLabel = mw.addStatusField(statusGroup, statGrid, "登录状态:", "未登录", 0)

	connStatus := "未连接"
	if ip, err := GetLocalIP(); err == nil && ip != "" {
		connStatus = "已连接"
	}
	mw.connectionLabel = mw.addStatusField(statusGroup, statGrid, "网络连接:", connStatus, 1)
	mw.ipLabel = mw.addStatusField(statusGroup, statGrid, "本地 IP:", "---", 2)
	mw.gatewayLabel = mw.addStatusField(statusGroup, statGrid, "网关地址:", "---", 3)
	mw.lastLoginLabel = mw.addStatusField(statusGroup, statGrid, "上次登录:", "---", 4)

	statGrid.SetColumnStretchFactor(0, 0)
	statGrid.SetColumnStretchFactor(1, 1)

	// --- Action Buttons ---
	btnRow, err := walk.NewComposite(mw)
	if err != nil {
		return err
	}
	btnRow.SetLayout(walk.NewHBoxLayout())

	mw.loginBtn, err = walk.NewPushButton(btnRow)
	if err != nil {
		return err
	}
	mw.loginBtn.SetText("登录 Login")

	mw.reconnectBtn, err = walk.NewPushButton(btnRow)
	if err != nil {
		return err
	}
	mw.reconnectBtn.SetText("重连 Reconnect")

	refreshNetBtn, err := walk.NewPushButton(btnRow)
	if err != nil {
		return err
	}
	refreshNetBtn.SetText("刷新网络")

	refreshIpBtn, err := walk.NewPushButton(btnRow)
	if err != nil {
		return err
	}
	refreshIpBtn.SetText("刷新 IP")

	advancedBtn, err := walk.NewPushButton(btnRow)
	if err != nil {
		return err
	}
	advancedBtn.SetText("高级设置...")

	// --- Log Panel ---
	logGroup, err := walk.NewGroupBox(mw)
	if err != nil {
		return err
	}
	logGroup.SetTitle("日志 Log")
	logGroup.SetLayout(walk.NewVBoxLayout())

	// Log level filter
	filterRow, err := walk.NewComposite(logGroup)
	if err != nil {
		return err
	}
	filterRow.SetLayout(walk.NewHBoxLayout())

	filterLabel, err := walk.NewLabel(filterRow)
	if err != nil {
		return err
	}
	filterLabel.SetText("级别:")

	mw.logLevelFilter, err = walk.NewComboBox(filterRow)
	if err != nil {
		return err
	}
	mw.logLevelFilter.SetModel([]string{"DEBUG", "INFO", "WARN", "ERROR"})
	mw.logLevelFilter.SetCurrentIndex(1)

	// Log text area (custom-painted with color-coded levels)
	mw.logView, err = NewLogView(logGroup)
	if err != nil {
		return err
	}
	mw.logView.SetMinMaxSize(walk.Size{Width: 480, Height: 100}, walk.Size{Width: 800, Height: 400})

	// Log buttons
	logBtnRow, err := walk.NewComposite(logGroup)
	if err != nil {
		return err
	}
	logBtnRow.SetLayout(walk.NewHBoxLayout())

	clearDnsBtn, err := walk.NewPushButton(logBtnRow)
	if err != nil {
		return err
	}
	clearDnsBtn.SetText("清除DNS缓存")

	clearLogBtn, err := walk.NewPushButton(logBtnRow)
	if err != nil {
		return err
	}
	clearLogBtn.SetText("清空日志")

	// GitHub link (bottom of log panel, unobtrusive)
	githubLink, err := walk.NewLinkLabel(logGroup)
	if err != nil {
		return err
	}
	githubLink.SetText(`<a>GitHub 开源 · 代码可审查</a>`)
	githubLink.LinkActivated().Attach(func(link *walk.LinkLabelLink) {
		openBrowser("https://github.com/Kurisut111na/CampusAutoLogin")
	})

	// --- Wire up events ---
	mw.showPasswordBtn.Clicked().Attach(mw.onTogglePassword)
	mw.loginBtn.Clicked().Attach(mw.onLoginClicked)
	mw.reconnectBtn.Clicked().Attach(mw.onReconnectClicked)
	refreshIpBtn.Clicked().Attach(mw.onRefreshIP)
	refreshNetBtn.Clicked().Attach(mw.onRefreshNetwork)
	advancedBtn.Clicked().Attach(mw.onAdvancedSettings)
	clearDnsBtn.Clicked().Attach(mw.onClearDNS)
	clearLogBtn.Clicked().Attach(mw.onClearLog)
	mw.logLevelFilter.CurrentIndexChanged().Attach(mw.onLogFilterChanged)

	// Close → minimize to tray
	mw.Closing().Attach(func(canceled *bool, reason walk.CloseReason) {
		*canceled = true
		mw.Hide()
	})

	return nil
}

func (mw *MainWindow) addStatusField(parent *walk.GroupBox, grid *walk.GridLayout, label string, initial string, row int) *walk.LineEdit {
	lbl, err := walk.NewLabel(parent)
	if err == nil {
		lbl.SetText(label)
		grid.SetRange(lbl, walk.Rectangle{X: 0, Y: row, Width: 1, Height: 1})
	}

	le, err := walk.NewLineEdit(parent)
	if err != nil {
		return nil
	}
	le.SetReadOnly(true)
	le.SetText(initial)
	grid.SetRange(le, walk.Rectangle{X: 1, Y: row, Width: 1, Height: 1})
	return le
}

// =============================================================================
// Config ↔ UI
// =============================================================================

func (mw *MainWindow) loadConfigToUI() {
	mw.loading = true
	defer func() { mw.loading = false }()

	mw.usernameEdit.SetText(mw.cfg.Username)
	if mw.cfg.RememberPassword && mw.cfg.Password != "" {
		mw.passwordEdit.SetText(mw.cfg.Password)
		mw.sessionPassword = mw.cfg.Password
	}

	opIndex := 0
	switch mw.cfg.Operator {
	case "unicom":
		opIndex = 1
	case "telecom":
		opIndex = 2
	case "campus":
		opIndex = 3
	}
	mw.operatorCombo.SetCurrentIndex(opIndex)

	mw.autoLoginCheck.SetChecked(mw.cfg.AutoLogin)
	mw.rememberPwdCheck.SetChecked(mw.cfg.RememberPassword)
	mw.heartbeatCheck.SetChecked(mw.cfg.HeartbeatEnabled)

	if mw.cfg.LastLoginTime != "" {
		mw.lastLoginLabel.SetText(mw.cfg.LastLoginTime)
	}

	mw.updateStatusDisplay()
}

func (mw *MainWindow) saveConfigFromUI() {
	if mw.loading {
		return
	}

	mw.cfg.Username = mw.usernameEdit.Text()
	mw.cfg.AutoLogin = mw.autoLoginCheck.Checked()
	mw.cfg.RememberPassword = mw.rememberPwdCheck.Checked()
	mw.cfg.HeartbeatEnabled = mw.heartbeatCheck.Checked()

	switch mw.operatorCombo.CurrentIndex() {
	case 0:
		mw.cfg.Operator = "cmcc"
	case 1:
		mw.cfg.Operator = "unicom"
	case 2:
		mw.cfg.Operator = "telecom"
	case 3:
		mw.cfg.Operator = "campus"
	}

	if mw.cfg.RememberPassword {
		mw.cfg.Password = mw.passwordEdit.Text()
		mw.sessionPassword = mw.cfg.Password
	} else {
		mw.cfg.Password = ""
		mw.sessionPassword = mw.passwordEdit.Text()
	}

	// Debounce save (500ms)
	if mw.saveTimer != nil {
		mw.saveTimer.Stop()
	}
	mw.saveTimer = time.AfterFunc(500*time.Millisecond, func() {
		if err := mw.configMgr.SaveConfig(mw.cfg); err != nil {
			GetLogger().Error("Failed to save config: %v", err)
		}
	})
}

func (mw *MainWindow) updateStatusDisplay() {
	netInfo := GetNetworkInfoFast()

	if netInfo.LocalIP != "" {
		mw.ipLabel.SetText(netInfo.LocalIP)
		mw.connectionLabel.SetText("已连接")
	} else {
		mw.ipLabel.SetText("未获取")
		mw.connectionLabel.SetText("未连接")
	}

	gateway := mw.cfg.CustomGateway
	if gateway != "" {
		mw.gatewayLabel.SetText(gateway)
	} else {
		mw.gatewayLabel.SetText("探测中...")
	}

	go func() {
		gw := ProbeGateway()
		mw.Synchronize(func() {
			if gw != "" {
				mw.gatewayLabel.SetText(gw)
			} else if mw.cfg.CustomGateway == "" {
				mw.gatewayLabel.SetText("未检测到")
			}
		})
	}()

	// Refine with real TCP reachability check (Bing)
	go func() {
		if CheckConnectivity() {
			mw.Synchronize(func() { mw.connectionLabel.SetText("已连接") })
		} else {
			mw.Synchronize(func() { mw.connectionLabel.SetText("未连接") })
		}
	}()
}

// =============================================================================
// Button Handlers
// =============================================================================

func (mw *MainWindow) onLoginClicked() {
	mw.saveConfigFromUI()

	username := mw.usernameEdit.Text()
	password := mw.passwordEdit.Text()
	if password == "" {
		password = mw.sessionPassword
	}
	operator := mw.cfg.Operator

	if username == "" || password == "" {
		walk.MsgBox(mw, "错误", "请输入用户名和密码", walk.MsgBoxIconError)
		return
	}

	mw.setLoginEnabled(false)
	mw.statusLabel.SetText("正在登录...")
	mw.appendLog(LogInfo, "Initiating login...")

	// All slow operations in goroutine: gateway probing + login HTTP requests
	go func() {
		gateway := mw.getGateway()
		netInfo := GetNetworkInfoFast()
		result := mw.loginMgr.Login(gateway, username, operator, password, netInfo.LocalIP, netInfo.MAC, netInfo.IPv6)
		mw.Synchronize(func() {
			mw.setLoginEnabled(true)
			if result.Success {
				mw.onLoginSuccess(result.Engine)
			} else {
				mw.onLoginFailed(result.Engine, result.Message)
			}
		})
	}()
}

func (mw *MainWindow) onReconnectClicked() {
	if mw.reconnectOnCooldown {
		mw.appendLog(LogWarning, "Reconnect on cooldown, please wait...")
		return
	}

	mw.reconnectOnCooldown = true
	mw.saveConfigFromUI()
	mw.onLoginClicked()

	// 2 second cooldown
	mw.reconnectCooldown = time.AfterFunc(2*time.Second, func() {
		mw.Synchronize(func() { mw.reconnectOnCooldown = false })
	})
}

func (mw *MainWindow) onRefreshIP() {
	mw.updateStatusDisplay()
	// Log message asynchronously to avoid blocking UI
	go func() {
		netInfo := GetNetworkInfo()
		mw.Synchronize(func() {
			if netInfo.LocalIP != "" {
				mw.appendLog(LogInfo, fmt.Sprintf("IP refreshed: %s", netInfo.LocalIP))
			}
		})
	}()
}

func (mw *MainWindow) onRefreshNetwork() {
	mw.updateStatusDisplay()
	// Log message asynchronously to avoid blocking UI
	go func() {
		netInfo := GetNetworkInfo()
		mw.Synchronize(func() {
			if netInfo.Gateway != "" {
				mw.appendLog(LogInfo, fmt.Sprintf("Gateway detected: %s (%s)", netInfo.Gateway, netInfo.NetType))
			} else {
				mw.appendLog(LogWarning, "Gateway not detected — network may be down")
			}
		})
	}()
}

func (mw *MainWindow) onAdvancedSettings() {
	mw.showAdvancedDialog()
}

func (mw *MainWindow) onClearDNS() {
	go func() {
		FlushDNS()
		FlushARP()
		mw.Synchronize(func() {
			mw.appendLog(LogDebug, "DNS/ARP cache flushed")
		})
	}()
}

func (mw *MainWindow) onClearLog() {
	GetLogger().CleanRecent()
	mw.logView.Clear()
	mw.appendLog(LogInfo, "Log cleared")
}

func (mw *MainWindow) onTogglePassword() {
	mw.passwordVisible = !mw.passwordVisible
	// Force redraw: EM_SETPASSWORDCHAR alone doesn't repaint existing text
	// Re-setting the text triggers a full repaint of the edit control
	currentText := mw.passwordEdit.Text()
	mw.passwordEdit.SetPasswordMode(!mw.passwordVisible)
	mw.passwordEdit.SetText("")
	mw.passwordEdit.SetText(currentText)
	if mw.passwordVisible {
		mw.showPasswordBtn.SetText("🙈") // password visible → show hide icon
	} else {
		mw.showPasswordBtn.SetText("👁") // password hidden → show reveal icon
	}
}

func (mw *MainWindow) onLogFilterChanged() {
	mw.refreshLogPanel()
}

// =============================================================================
// Login Result Handlers
// =============================================================================

func (mw *MainWindow) onLoginSuccess(engine string) {
	mw.statusLabel.SetText(fmt.Sprintf("已登录 (%s)", engine))
	mw.connectionLabel.SetText("已连接")
	mw.lastLoginLabel.SetText(time.Now().Format("2006-01-02 15:04:05"))
	mw.appendLog(LogInfo, fmt.Sprintf("Login successful via %s", engine))

	mw.tray.SetLoggedIn(true)
	mw.tray.SetConnectionStatus(true)
	mw.tray.SetVisible(true)
	mw.tray.ShowBalloon("校园网登录成功", fmt.Sprintf("已通过 %s 完成认证，可以上网了", engine))

	// Start heartbeat
	if mw.cfg.HeartbeatEnabled {
		mw.heartbeat.SetInterval(mw.cfg.HeartbeatInterval)
		mw.heartbeat.Start()
	}

	// Start DNS warmup
	mw.startWarmup()

	// Save config
	mw.saveConfigFromUI()
}

func (mw *MainWindow) onLoginFailed(engine, errMsg string) {
	mw.statusLabel.SetText("登录失败")
	mw.appendLog(LogError, fmt.Sprintf("Login failed [%s]: %s", engine, errMsg))

	mw.tray.SetLoggedIn(false)
	mw.tray.SetConnectionStatus(false)
	mw.tray.ShowBalloon("校园网登录失败", errMsg)
}

// =============================================================================
// DNS Warmup
// =============================================================================

func (mw *MainWindow) startWarmup() {
	client := &http.Client{Timeout: 10 * time.Second}

	go func() {
		steps := []struct {
			delay time.Duration
			url   string
			desc  string
		}{
			{0, "https://www.baidu.com", "Baidu"},
			{3 * time.Second, "https://www.bing.com", "Bing"},
			{5 * time.Second, "", "Gateway"},
			{6 * time.Second, "", "Final DNS flush"},
		}
		prevDelay := time.Duration(0)
		for _, step := range steps {
			time.Sleep(step.delay - prevDelay)
			prevDelay = step.delay
			FlushDNS()
			FlushARP()
			if step.url != "" {
				resp, err := client.Get(step.url)
				if err == nil {
					resp.Body.Close()
				}
			}
			mw.Synchronize(func() {
				mw.appendLog(LogDebug, fmt.Sprintf("Warmup: %s done", step.desc))
			})
		}
	}()
}

// =============================================================================
// Advanced Settings Dialog
// =============================================================================

func (mw *MainWindow) showAdvancedDialog() {
	// Ensure the main window is visible before showing a child dialog.
	// If the window was hidden (minimized to tray), Walk uses the owner's
	// bounds to center the dialog — hidden windows give garbage coordinates,
	// causing the dialog to appear off-screen or vanish immediately.
	mw.Show()

	dlg, err := walk.NewDialog(mw)
	if err != nil {
		return
	}

	dlg.SetTitle("高级设置 Advanced Settings")
	dlgW, dlgH := 500, 800
	dlg.SetSize(walk.Size{Width: dlgW, Height: dlgH})

	dlg.SetLayout(walk.NewVBoxLayout())

	// ==========================================================================
	// Heartbeat section
	// ==========================================================================
	hbGroup, _ := walk.NewGroupBox(dlg)
	hbGroup.SetTitle("心跳检测 Heartbeat")
	hbGrid := walk.NewGridLayout()
	hbGrid.SetColumnStretchFactor(0, 0) // label column: fixed
	hbGrid.SetColumnStretchFactor(1, 1) // control column: stretches
	hbGrid.SetSpacing(4)
	hbGroup.SetLayout(hbGrid)

	// Row 0: enable checkbox
	hbEnableLabel, _ := walk.NewLabel(hbGroup)
	hbEnableLabel.SetText("启用心跳:")
	hbEnableLabel.SetMinMaxSize(walk.Size{Width: 80, Height: 0}, walk.Size{Width: 80, Height: 24})
	hbGrid.SetRange(hbEnableLabel, walk.Rectangle{X: 0, Y: 0, Width: 1, Height: 1})
	hbEnable, _ := walk.NewCheckBox(hbGroup)
	hbEnable.SetText("断线自动重连")
	hbEnable.SetChecked(mw.cfg.HeartbeatEnabled)
	hbGrid.SetRange(hbEnable, walk.Rectangle{X: 1, Y: 0, Width: 1, Height: 1})

	// Row 1: interval (compact: label + input + "秒")
	hbIntLabel, _ := walk.NewLabel(hbGroup)
	hbIntLabel.SetText("检测间隔:")
	hbIntLabel.SetMinMaxSize(walk.Size{Width: 80, Height: 0}, walk.Size{Width: 80, Height: 24})
	hbGrid.SetRange(hbIntLabel, walk.Rectangle{X: 0, Y: 1, Width: 1, Height: 1})

	// Wrap number input + "秒" label in a horizontal composite
	intRow, _ := walk.NewComposite(hbGroup)
	intRow.SetLayout(walk.NewHBoxLayout())
	hbInterval, _ := walk.NewLineEdit(intRow)
	hbInterval.SetText(fmt.Sprintf("%d", mw.cfg.HeartbeatInterval))
	hbInterval.SetMaxLength(3)
	hbInterval.SetMinMaxSize(walk.Size{Width: 36, Height: 24}, walk.Size{Width: 44, Height: 24})
	intUnit, _ := walk.NewLabel(intRow)
	intUnit.SetText("秒 (范围 15-300)")
	intUnit.SetTextColor(walk.RGB(128, 128, 128))
	hbGrid.SetRange(intRow, walk.Rectangle{X: 1, Y: 1, Width: 1, Height: 1})

	// Row 2: ping URLs
	hbUrlLabel, _ := walk.NewLabel(hbGroup)
	hbUrlLabel.SetText("Ping URLs:")
	hbUrlLabel.SetMinMaxSize(walk.Size{Width: 80, Height: 0}, walk.Size{Width: 80, Height: 24})
	hbGrid.SetRange(hbUrlLabel, walk.Rectangle{X: 0, Y: 2, Width: 1, Height: 1})
	pingEdit, _ := walk.NewTextEdit(hbGroup)
	pingEdit.SetMinMaxSize(walk.Size{Width: 360, Height: 60}, walk.Size{Width: 400, Height: 100})
	hbGrid.SetRange(pingEdit, walk.Rectangle{X: 1, Y: 2, Width: 1, Height: 1})
	pingText := ""
	for _, u := range mw.cfg.PingURLs {
		pingText += u + "\r\n"
	}
	pingEdit.SetText(pingText)

	// ==========================================================================
	// Startup section
	// ==========================================================================
	autoGroup, _ := walk.NewGroupBox(dlg)
	autoGroup.SetTitle("启动 Startup")
	autoGroup.SetLayout(walk.NewVBoxLayout())

	autoStartCheck, _ := walk.NewCheckBox(autoGroup)
	autoStartCheck.SetText("开机自启动 (Auto-start with Windows)")
	autoStartCheck.SetChecked(mw.cfg.AutoStart)

	minimizedWarning, _ := walk.NewLabel(autoGroup)
	minimizedWarning.SetText("  └ 注册表项: HKCU\\...\\Run\\CampusAutoLogin")
	minimizedWarning.SetTextColor(walk.RGB(128, 128, 128))

	minimizedCheck, _ := walk.NewCheckBox(autoGroup)
	minimizedCheck.SetText("启动时最小化到托盘 (Start minimized)")
	minimizedCheck.SetChecked(mw.cfg.StartMinimized)

	// ==========================================================================
	// Custom Gateway section
	// ==========================================================================
	gwGroup, _ := walk.NewGroupBox(dlg)
	gwGroup.SetTitle("网关 Gateway")
	gwGrid := walk.NewGridLayout()
	gwGrid.SetColumnStretchFactor(0, 0)
	gwGrid.SetColumnStretchFactor(1, 1)
	gwGroup.SetLayout(gwGrid)

	gwLabel, _ := walk.NewLabel(gwGroup)
	gwLabel.SetText("自定义网关:")
	gwLabel.SetMinMaxSize(walk.Size{Width: 80, Height: 0}, walk.Size{Width: 80, Height: 24})
	gwGrid.SetRange(gwLabel, walk.Rectangle{X: 0, Y: 0, Width: 1, Height: 1})
	gwEdit, _ := walk.NewLineEdit(gwGroup)
	gwEdit.SetText(mw.cfg.CustomGateway)
	gwGrid.SetRange(gwEdit, walk.Rectangle{X: 1, Y: 0, Width: 1, Height: 1})

	// ==========================================================================
	// Log dir info
	// ==========================================================================
	logGroup, _ := walk.NewGroupBox(dlg)
	logGroup.SetLayout(walk.NewVBoxLayout())
	logDirLabel, _ := walk.NewLabel(logGroup)
	logDirLabel.SetText(fmt.Sprintf("日志目录: %s", GetLogger().LogDir()))

	// ==========================================================================
	// Support / Donation Section
	// ==========================================================================
	supportGroup, _ := walk.NewGroupBox(dlg)
	supportGroup.SetTitle("赞赏支持 Support")
	supportGroup.SetLayout(walk.NewVBoxLayout())

	// Decode the embedded QR code image (supports PNG, JPEG, etc.)
	// Original image is 1152×1152 — render at 280×280 for reliable phone scanning.
	img, _, err := image.Decode(bytes.NewReader(qrCodePNG))
	if err == nil {
		if bmp, err := walk.NewBitmapFromImage(img); err == nil {
			iv, err := walk.NewImageView(supportGroup)
			if err == nil {
				iv.SetImage(bmp)
				iv.SetMode(walk.ImageViewModeShrink)
				iv.SetSize(walk.Size{Width: 280, Height: 280})
				iv.SetMinMaxSize(walk.Size{Width: 240, Height: 240}, walk.Size{Width: 320, Height: 320})
			}
		}
	} else {
		GetLogger().Warn("Failed to decode embedded QR code image: %v", err)
	}

	supportLabel, _ := walk.NewLabel(supportGroup)
	supportLabel.SetText("本软件永久免费。赞赏码是支持开发者的唯一渠道。\n如遇任何形式的付费下载，请谨防受骗。")
	supportLabel.SetTextAlignment(walk.AlignCenter)
	supportLabel.SetTextColor(walk.RGB(100, 100, 100))

	// ==========================================================================
	// OK / Cancel buttons
	// ==========================================================================
	btnRow, _ := walk.NewComposite(dlg)
	btnRow.SetLayout(walk.NewHBoxLayout())

	// Spacer pushes buttons to the right
	spacer, _ := walk.NewLabel(btnRow)
	spacer.SetText("")

	saveBtn, _ := walk.NewPushButton(btnRow)
	saveBtn.SetText("保存 Save")
	cancelBtn, _ := walk.NewPushButton(btnRow)
	cancelBtn.SetText("取消 Cancel")

	// ==========================================================================
	// Save handler
	// ==========================================================================
	saveBtn.Clicked().Attach(func() {
		mw.cfg.HeartbeatEnabled = hbEnable.Checked()
		// Parse interval from left-aligned LineEdit
		interval := mw.cfg.HeartbeatInterval // keep current as fallback
		if v, err := fmt.Sscanf(hbInterval.Text(), "%d", &interval); err == nil && v == 1 {
			if interval < 15 {
				interval = 15
			} else if interval > 300 {
				interval = 300
			}
		}
		mw.cfg.HeartbeatInterval = interval

		// Auto-start: always sync registry to match checkbox state
		newAutoStart := autoStartCheck.Checked()
		if newAutoStart {
			if err := RegisterAutoStart(); err != nil {
				GetLogger().Error("Auto-start register failed: %v", err)
				walk.MsgBox(dlg, "开机自启动设置失败",
					fmt.Sprintf("无法写入注册表:\n%s\n\n请尝试以管理员身份运行本程序。", err.Error()),
					walk.MsgBoxIconError)
			}
		} else {
			if err := UnregisterAutoStart(); err != nil {
				GetLogger().Error("Auto-start unregister failed: %v", err)
			}
		}
		mw.cfg.AutoStart = newAutoStart
		mw.cfg.StartMinimized = minimizedCheck.Checked()
		mw.cfg.CustomGateway = gwEdit.Text()

		// Parse URLs
		urls := []string{}
		for _, line := range splitLines(pingEdit.Text()) {
			line = trimSpace(line)
			if line != "" {
				urls = append(urls, line)
			}
		}
		if len(urls) > 0 {
			mw.cfg.PingURLs = urls
		}

		// Apply immediately
		mw.heartbeatCheck.SetChecked(mw.cfg.HeartbeatEnabled)
		mw.heartbeat.SetInterval(mw.cfg.HeartbeatInterval)
		mw.heartbeat.SetURLs(mw.cfg.PingURLs)

		mw.saveConfigFromUI()
		mw.appendLog(LogInfo, "Advanced settings saved")
		dlg.Accept()
	})

	cancelBtn.Clicked().Attach(func() {
		dlg.Cancel()
	})

	dlg.Run()
}

// =============================================================================
// Callbacks setup (connected to heartbeat/tray)
// =============================================================================

func (mw *MainWindow) setupCallbacks() {
	mw.heartbeat.OnConnected(func() {
		mw.Synchronize(func() {
			mw.connectionLabel.SetText("已连接")
			mw.appendLog(LogDebug, "Heartbeat: connection alive")
			// Only notify on recovery (when tray was in disconnected state)
			if !mw.tray.connected {
				mw.tray.SetConnectionStatus(true)
				mw.tray.ShowBalloon("校园网恢复", "网络连接已恢复")
			}
		})
	})

	mw.heartbeat.OnLost(func() {
		mw.Synchronize(func() {
			mw.connectionLabel.SetText("已断开")
			mw.appendLog(LogWarning, "Heartbeat: connection LOST!")
			mw.tray.SetConnectionStatus(false)
			mw.tray.ShowBalloon("校园网断开", "网络连接已断开，正在尝试重连...")
		})
	})

	mw.heartbeat.OnReconnectRequested(func() {
		mw.Synchronize(func() {
			mw.statusLabel.SetText("正在自动重连...")
			mw.appendLog(LogInfo, "Heartbeat: attempting reconnect...")
			mw.onReconnectClicked()
		})
	})

	mw.tray.OnShowWindow(func() {
		mw.Synchronize(func() {
			if mw.Visible() {
				mw.Hide()
			} else {
				mw.Show()
			}
		})
	})

	mw.tray.OnReconnect(func() {
		mw.Synchronize(func() {
			mw.onReconnectClicked()
		})
	})

	mw.tray.OnOpenLogDir(func() {
		logDir := GetLogger().LogDir()
		exec.Command("explorer", logDir).Start()
	})

	mw.tray.OnQuit(func() {
		mw.Synchronize(func() {
			mw.saveConfigFromUI()
			mw.heartbeat.Stop()
			GetLogger().Close()
			walk.App().Exit(0)
		})
	})

	// Input change → debounce save
	mw.usernameEdit.TextChanged().Attach(func() { mw.saveConfigFromUI() })
	mw.passwordEdit.TextChanged().Attach(func() { mw.saveConfigFromUI() })
	mw.operatorCombo.CurrentIndexChanged().Attach(func() { mw.saveConfigFromUI() })
	mw.autoLoginCheck.CheckStateChanged().Attach(func() { mw.saveConfigFromUI() })
	mw.rememberPwdCheck.CheckStateChanged().Attach(func() { mw.saveConfigFromUI() })
	mw.heartbeatCheck.CheckStateChanged().Attach(func() { mw.saveConfigFromUI() })
}

// =============================================================================
// Auto-login (called after startup delay)
// =============================================================================

func (mw *MainWindow) ShouldStartMinimized() bool {
	return mw.cfg.StartMinimized && mw.cfg.AutoLogin &&
		mw.cfg.Username != "" && mw.cfg.Password != ""
}

func (mw *MainWindow) TriggerAutoLogin() {
	if !mw.cfg.AutoLogin || mw.cfg.Username == "" || mw.cfg.Password == "" {
		GetLogger().Info("Auto-login skipped (disabled or no credentials)")
		return
	}

	GetLogger().Info("Auto-login triggered after startup delay")
	go func() {
		time.Sleep(3 * time.Second)

		// Retry gateway probe up to 15 times (3s each)
		for i := 0; i < 15; i++ {
			gw := ProbeGateway()
			if gw != "" {
				mw.cfg.CustomGateway = gw
				break
			}
			time.Sleep(3 * time.Second)
		}

		mw.Synchronize(func() {
			mw.onLoginClicked()
		})
	}()
}

// =============================================================================
// Helpers
// =============================================================================

func (mw *MainWindow) getGateway() string {
	if mw.cfg.CustomGateway != "" {
		return mw.cfg.CustomGateway
	}
	gw := ProbeGateway()
	if gw != "" {
		return gw
	}
	// Last resort fallback
	return "1.2.3.4"
}

func (mw *MainWindow) setLoginEnabled(enabled bool) {
	mw.loginBtn.SetEnabled(enabled)
	mw.reconnectBtn.SetEnabled(enabled)
}

func (mw *MainWindow) appendLog(level LogLevel, message string) {
	mw.logView.Append(level, message)
}

func (mw *MainWindow) refreshLogPanel() {
	level := LogInfo
	switch mw.logLevelFilter.CurrentIndex() {
	case 0:
		level = LogDebug
	case 1:
		level = LogInfo
	case 2:
		level = LogWarning
	case 3:
		level = LogError
	}

	allEntries := GetLogger().AllRecentEntries()
	filtered := make([]LogEntry, 0, len(allEntries))
	for _, e := range allEntries {
		if e.Level >= level {
			filtered = append(filtered, e)
		}
	}
	mw.logView.SetEntries(filtered)
}
