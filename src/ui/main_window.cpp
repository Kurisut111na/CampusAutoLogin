#include "main_window.h"
#include "tray_icon.h"
#include "auto_start.h"
#include "advanced_settings_dialog.h"

#include "config_manager.h"
#include "ip_detector.h"
#include "login_manager.h"
#include "heartbeat.h"
#include "network_manager.h"
#include "logger.h"

#include <QVBoxLayout>
#include <QHBoxLayout>
#include <QFormLayout>
#include <QGroupBox>
#include <QMessageBox>
#include <QCloseEvent>
#include <QApplication>
#include <QProcess>
#include <QStyle>
#include <QScrollBar>
#include <QDesktopServices>
#include <QUrl>
#include <QNetworkAccessManager>
#include <QNetworkRequest>
#include <QClipboard>
#include <QCheckBox>

MainWindow::MainWindow(QWidget* parent)
    : QMainWindow(parent)
    , m_configManager(new ConfigManager(this))
    , m_ipDetector(new IpDetector(this))
    , m_loginManager(new LoginManager(this))
    , m_heartbeat(new Heartbeat(this))
    , m_networkManager(new NetworkManager(m_ipDetector, this))
    , m_trayIcon(new TrayIcon(this))
    , m_autoStartManager(new AutoStartManager(this))
    , m_advancedDialog(new AdvancedSettingsDialog(this))
    , m_currentConfig(new AppConfig)
    , m_autoLoginTimer(new QTimer(this))
    , m_reconnectCooldown(new QTimer(this))
    , m_dnsDelayTimer(new QTimer(this))
    , m_warmupTimer(new QTimer(this))
    , m_saveTimer(new QTimer(this))
{
    setupUi();
    setupConnections();
    loadSettings();

    m_autoLoginTimer->setSingleShot(true);
    m_autoLoginTimer->setInterval(3000);  // 3s delay — let network card finish initializing after boot
    connect(m_autoLoginTimer, &QTimer::timeout, this, &MainWindow::onAutoLoginTriggered);

    m_reconnectCooldown->setSingleShot(true);
    m_reconnectCooldown->setInterval(2000);
    connect(m_reconnectCooldown, &QTimer::timeout, this, &MainWindow::onReconnectCooldownTimeout);

    m_dnsDelayTimer->setSingleShot(true);
    m_dnsDelayTimer->setInterval(1000);
    connect(m_dnsDelayTimer, &QTimer::timeout, this, &MainWindow::flushDns);

    m_warmupTimer->setSingleShot(true);
    connect(m_warmupTimer, &QTimer::timeout, this, [this]() {
        m_warmupCount++;
        if (m_warmupCount > 4)
            return;

        flushDns();
        flushArp();

        // Send one real HTTP request per round to force TCP path through the newly-authenticated gateway.
        // 4 rounds, 3 HTTP requests total — less than opening 3 web pages by hand.
        QNetworkAccessManager* mgr = new QNetworkAccessManager(this);
        connect(mgr, &QNetworkAccessManager::finished, mgr, &QNetworkAccessManager::deleteLater);
        if (m_warmupCount == 1) {
            mgr->get(QNetworkRequest(QUrl("https://www.baidu.com")));
        } else if (m_warmupCount == 2) {
            mgr->get(QNetworkRequest(QUrl("https://www.bing.com")));
        } else if (m_warmupCount == 3) {
            mgr->get(QNetworkRequest(QUrl("http://" + m_networkManager->currentGateway() + "/")));
        }
        // Round 4: DNS/ARP only, no HTTP

        static const int delays[] = {0, 3000, 5000, 6000, 7000};
        m_warmupTimer->start(delays[m_warmupCount]);
    });

    m_saveTimer->setSingleShot(true);
    m_saveTimer->setInterval(500);
    connect(m_saveTimer, &QTimer::timeout, this, &MainWindow::saveSettings);

    Logger::instance().info("Program started");
}

MainWindow::~MainWindow()
{
    delete m_currentConfig;
}

bool MainWindow::shouldStartMinimized() const
{
    // Only minimize when the user has explicitly opted in AND auto-login is
    // actually configured (otherwise they need to see the window to set up).
    return m_currentConfig->startMinimized
        && m_currentConfig->autoLogin
        && !m_currentConfig->username.isEmpty();
}

void MainWindow::setupUi()
{
    setWindowTitle("校园网自动登录");
    setMinimumSize(520, 680);

    QWidget* centralWidget = new QWidget(this);
    setCentralWidget(centralWidget);

    QVBoxLayout* mainLayout = new QVBoxLayout(centralWidget);
    mainLayout->setSpacing(10);
    mainLayout->setContentsMargins(12, 12, 12, 12);

    QGroupBox* accountGroup = new QGroupBox("账号");
    QFormLayout* accountLayout = new QFormLayout(accountGroup);

    m_usernameEdit = new QLineEdit;
    m_usernameEdit->setPlaceholderText("学号");
    m_usernameEdit->setToolTip("输入校园网用户名（学号）");
    accountLayout->addRow("用户名：", m_usernameEdit);

    QHBoxLayout* passwordLayout = new QHBoxLayout;
    m_passwordEdit = new QLineEdit;
    m_passwordEdit->setEchoMode(QLineEdit::Password);
    m_passwordEdit->setPlaceholderText("密码");
    m_showPasswordBtn = new QPushButton;
    m_showPasswordBtn->setIcon(style()->standardIcon(QStyle::SP_FileDialogContentsView));
    m_showPasswordBtn->setFixedWidth(32);
    m_showPasswordBtn->setToolTip("显示/隐藏密码");
    passwordLayout->addWidget(m_passwordEdit);
    passwordLayout->addWidget(m_showPasswordBtn);
    accountLayout->addRow("密码：", passwordLayout);

    m_operatorCombo = new QComboBox;
    m_operatorCombo->addItem("中国移动", "cmcc");
    m_operatorCombo->addItem("中国电信", "ctcc");
    m_operatorCombo->addItem("中国联通", "cucc");
    accountLayout->addRow("运营商：", m_operatorCombo);

    mainLayout->addWidget(accountGroup);

    QGroupBox* optionsGroup = new QGroupBox("选项");
    QVBoxLayout* optionsLayout = new QVBoxLayout(optionsGroup);

    m_autoLoginCheck = new QCheckBox("启动时自动登录");
    optionsLayout->addWidget(m_autoLoginCheck);

    m_rememberPasswordCheck = new QCheckBox("记住密码");
    m_rememberPasswordCheck->setChecked(true);
    optionsLayout->addWidget(m_rememberPasswordCheck);

    m_heartbeatCheck = new QCheckBox("启用心跳检测（定期检查网络连通性）");
    m_heartbeatCheck->setChecked(true);
    optionsLayout->addWidget(m_heartbeatCheck);

    mainLayout->addWidget(optionsGroup);

    QGroupBox* statusGroup = new QGroupBox("状态");
    QFormLayout* statusLayout = new QFormLayout(statusGroup);

    m_statusLabel = new QLabel("未登录");
    m_statusLabel->setStyleSheet("font-weight: bold; color: #f44336;");
    statusLayout->addRow("登录状态：", m_statusLabel);

    m_connectionLabel = new QLabel("检测中...");
    statusLayout->addRow("网络连接：", m_connectionLabel);

    m_ipLabel = new QLabel("未知");
    statusLayout->addRow("本地 IP：", m_ipLabel);

    m_gatewayLabel = new QLabel("探测中...");
    statusLayout->addRow("网关地址：", m_gatewayLabel);

    m_lastLoginLabel = new QLabel("从未");
    statusLayout->addRow("上次登录：", m_lastLoginLabel);

    mainLayout->addWidget(statusGroup);

    QHBoxLayout* actionLayout = new QHBoxLayout;

    m_loginBtn = new QPushButton("登录");
    m_loginBtn->setStyleSheet(
        "QPushButton { background-color: #4CAF50; color: white; padding: 8px 14px; "
        "border-radius: 4px; font-weight: bold; }"
        "QPushButton:hover { background-color: #45a049; }");
    actionLayout->addWidget(m_loginBtn);

    m_reconnectBtn = new QPushButton("重连");
    m_reconnectBtn->setEnabled(false);
    m_reconnectBtn->setStyleSheet(
        "QPushButton { background-color: #FF9800; color: white; padding: 8px 14px; "
        "border-radius: 4px; font-weight: bold; }"
        "QPushButton:hover { background-color: #F57C00; }");
    actionLayout->addWidget(m_reconnectBtn);

    mainLayout->addLayout(actionLayout);

    QHBoxLayout* configLayout = new QHBoxLayout;
    m_refreshNetworkBtn = new QPushButton("刷新网络连接");
    m_refreshNetworkBtn->setToolTip("手动刷新 DNS 缓存、ARP 表和网关连接状态");
    configLayout->addWidget(m_refreshNetworkBtn);

    m_refreshIpBtn = new QPushButton("刷新 IP");
    configLayout->addWidget(m_refreshIpBtn);

    m_advancedBtn = new QPushButton("高级设置...");
    configLayout->addWidget(m_advancedBtn);

    mainLayout->addLayout(configLayout);

    QGroupBox* logGroup = new QGroupBox("日志");
    QVBoxLayout* logLayout = new QVBoxLayout(logGroup);

    QHBoxLayout* logHeaderLayout = new QHBoxLayout;
    m_logLevelFilter = new QComboBox;
    m_logLevelFilter->addItem("全部", -1);
    m_logLevelFilter->addItem("INFO", static_cast<int>(LogLevel::Info));
    m_logLevelFilter->addItem("WARN", static_cast<int>(LogLevel::Warning));
    m_logLevelFilter->addItem("ERROR", static_cast<int>(LogLevel::Error));
    logHeaderLayout->addWidget(new QLabel("过滤："));
    logHeaderLayout->addWidget(m_logLevelFilter);

    m_clearChromeDnsBtn = new QPushButton("清理浏览器DNS缓存");
    m_clearChromeDnsBtn->setToolTip("登录后某些网站加载慢可能是浏览器内部DNS缓存导致的。点击打开chrome://net-internals页面，再点「Clear host cache」即可。");
    m_clearChromeDnsBtn->setStyleSheet("QPushButton { color: #2196F3; font-size: 11px; }");
    logHeaderLayout->addWidget(m_clearChromeDnsBtn);

    logHeaderLayout->addStretch();
    m_clearLogBtn = new QPushButton("清空");
    logHeaderLayout->addWidget(m_clearLogBtn);
    logLayout->addLayout(logHeaderLayout);

    m_logPanel = new QTextEdit;
    m_logPanel->setReadOnly(true);
    m_logPanel->setMaximumHeight(160);
    m_logPanel->setStyleSheet("QTextEdit { font-family: Consolas, monospace; font-size: 12px; }");
    logLayout->addWidget(m_logPanel);

    mainLayout->addWidget(logGroup);
}

void MainWindow::setupConnections()
{
    connect(m_showPasswordBtn, &QPushButton::clicked, this, [this]() {
        m_passwordVisible = !m_passwordVisible;
        m_passwordEdit->setEchoMode(m_passwordVisible ? QLineEdit::Normal : QLineEdit::Password);
    });

    // Sync credentials to config in real-time and debounce-save
    connect(m_usernameEdit, &QLineEdit::textChanged, this, [this](const QString& text) {
        m_currentConfig->username = text;
        if (!m_loading) m_saveTimer->start();
    });
    connect(m_passwordEdit, &QLineEdit::textChanged, this, [this](const QString& text) {
        if (m_currentConfig->rememberPassword) {
            m_currentConfig->password = text;
        }
        if (!m_loading) m_saveTimer->start();
    });

    connect(m_loginBtn, &QPushButton::clicked, this, &MainWindow::onLoginClicked);
    connect(m_reconnectBtn, &QPushButton::clicked, this, &MainWindow::onReconnectClicked);
    connect(m_refreshIpBtn, &QPushButton::clicked, this, &MainWindow::onRefreshIp);
    connect(m_refreshNetworkBtn, &QPushButton::clicked, this, &MainWindow::onRefreshNetwork);
    connect(m_advancedBtn, &QPushButton::clicked, this, &MainWindow::onAdvancedSettings);
    connect(m_clearLogBtn, &QPushButton::clicked, this, [this]() {
        m_logPanel->clear();
        Logger::instance().clearRecentEntries();
    });

    connect(m_logLevelFilter, QOverload<int>::of(&QComboBox::currentIndexChanged),
            this, [this](int) {
        int filterLevel = m_logLevelFilter->currentData().toInt();
        m_logPanel->clear();
        QList<LogEntry> entries = Logger::instance().recentEntries(50);
        for (const auto& entry : entries) {
            if (filterLevel == -1 || static_cast<int>(entry.level) >= filterLevel) {
                onNewLogEntry(entry);
            }
        }
    });

    connect(m_loginManager, &LoginManager::loginSuccess, this, &MainWindow::onLoginSuccess);
    connect(m_loginManager, &LoginManager::loginFailed, this, &MainWindow::onLoginFailed);

    connect(m_heartbeatCheck, &QCheckBox::toggled, this, [this](bool checked) {
        m_currentConfig->heartbeatEnabled = checked;
        if (checked) {
            m_heartbeat->setCheckUrls(m_currentConfig->pingUrls);
            m_heartbeat->start(m_currentConfig->heartbeatInterval);
        } else {
            m_heartbeat->stop();
        }
        if (!m_loading) m_configManager->saveConfig(*m_currentConfig);
    });

    connect(m_autoLoginCheck, &QCheckBox::toggled, this, [this](bool checked) {
        m_currentConfig->autoLogin = checked;
        if (!m_loading) m_configManager->saveConfig(*m_currentConfig);
    });

    connect(m_rememberPasswordCheck, &QCheckBox::toggled, this, [this](bool checked) {
        m_currentConfig->rememberPassword = checked;
        if (!checked) {
            m_currentConfig->password.clear();
        } else if (!m_passwordEdit->text().isEmpty()) {
            m_currentConfig->password = m_passwordEdit->text();
        }
        if (!m_loading) m_configManager->saveConfig(*m_currentConfig);
    });

    connect(m_operatorCombo, QOverload<int>::of(&QComboBox::currentIndexChanged),
            this, [this](int) {
        m_currentConfig->operator_ = m_operatorCombo->currentData().toString();
        if (!m_loading) m_configManager->saveConfig(*m_currentConfig);
    });

    connect(m_heartbeat, &Heartbeat::connectionLost, this, &MainWindow::onConnectionLost);
    connect(m_heartbeat, &Heartbeat::connectionRestored, this, &MainWindow::onConnectionRestored);
    connect(m_heartbeat, &Heartbeat::connectionAlive, this, &MainWindow::onConnectionAlive);

    connect(m_networkManager, &NetworkManager::gatewayReady, this, &MainWindow::onGatewayReady);
    connect(m_networkManager, &NetworkManager::gatewayProbeFailed, this, &MainWindow::onGatewayProbeFailed);
    connect(m_networkManager, &NetworkManager::gatewayChanged, this, &MainWindow::onGatewayChanged);

    connect(m_ipDetector, &IpDetector::networkTypeChanged, this, &MainWindow::onNetworkTypeChanged);

    connect(&Logger::instance(), &Logger::newLogEntry, this, &MainWindow::onNewLogEntry);

    connect(m_trayIcon, &TrayIcon::activated, this, &MainWindow::onTrayActivated);
    connect(m_trayIcon, &TrayIcon::showRequested, this, [this]() {
        show();
        raise();
        activateWindow();
    });
    connect(m_trayIcon, &TrayIcon::reconnectRequested, this, [this]() {
        show();
        raise();
        activateWindow();
        onReconnectClicked();
    });
    connect(m_trayIcon, &TrayIcon::openLogDirRequested, this, [this]() {
        QDesktopServices::openUrl(QUrl::fromLocalFile(Logger::instance().logDir()));
    });
    connect(m_clearChromeDnsBtn, &QPushButton::clicked, this, [this]() {
        QApplication::clipboard()->setText("chrome://net-internals/#dns");
        QMessageBox box(this);
        box.setWindowTitle("清理浏览器DNS缓存");
        box.setText("已复制链接到剪贴板。\n\n"
                    "1. 打开 Chrome 浏览器\n"
                    "2. 在地址栏粘贴并回车\n"
                    "3. 点击「DNS」→「Clear host cache」");
        box.setIcon(QMessageBox::Information);
        box.setStandardButtons(QMessageBox::Ok);
        box.button(QMessageBox::Ok)->setText("知道了");
        QCheckBox* cb = new QCheckBox("不再提示");
        box.setCheckBox(cb);
        box.exec();
        if (cb->isChecked()) {
            m_clearChromeDnsBtn->setVisible(false);
        }
    });

    connect(m_trayIcon, &TrayIcon::quitRequested, this, [this]() {
        m_heartbeat->stop();
        m_loginManager->abortLogin();
        saveSettings();
        QApplication::quit();
    });
}

void MainWindow::loadSettings()
{
    m_loading = true;
    bool loaded = m_configManager->loadConfig(*m_currentConfig);

    if (loaded) {
        m_usernameEdit->setText(m_currentConfig->username);
        m_autoLoginCheck->setChecked(m_currentConfig->autoLogin);
        m_rememberPasswordCheck->setChecked(m_currentConfig->rememberPassword);
        m_heartbeatCheck->setChecked(m_currentConfig->heartbeatEnabled);

        if (m_currentConfig->rememberPassword) {
            m_passwordEdit->setText(m_currentConfig->password);
        }

        int opIndex = m_operatorCombo->findData(m_currentConfig->operator_);
        if (opIndex >= 0)
            m_operatorCombo->setCurrentIndex(opIndex);

        if (m_currentConfig->lastLoginTime.isValid()) {
            m_lastLoginLabel->setText(m_currentConfig->lastLoginTime.toString("yyyy-MM-dd hh:mm:ss"));
        }

        if (!m_currentConfig->customGateway.isEmpty()) {
            m_networkManager->setCustomGateway(m_currentConfig->customGateway);
        }

        m_heartbeat->setInterval(m_currentConfig->heartbeatInterval);
        if (!m_currentConfig->pingUrls.isEmpty())
            m_heartbeat->setCheckUrls(m_currentConfig->pingUrls);
        Logger::instance().info("Settings loaded from config file");
    }

    QString ip = m_ipDetector->detectLocalIp();
    m_ipLabel->setText(ip.isEmpty() ? "Unknown" : ip);

    if (m_currentConfig->customGateway.isEmpty()) {
        m_networkManager->probeGateway();
    }

    if (m_currentConfig->heartbeatEnabled)
        m_heartbeat->start(m_currentConfig->heartbeatInterval);

    if (m_currentConfig->autoLogin && !m_currentConfig->username.isEmpty()) {
        m_autoLoginTimer->start();
    }

    m_loading = false;
}

void MainWindow::saveSettings()
{
    m_configManager->saveConfig(*m_currentConfig);
}

void MainWindow::updateStatusDisplay()
{
    if (m_loginManager->isLoggedIn()) {
        m_statusLabel->setText("已连接");
        m_statusLabel->setStyleSheet("font-weight: bold; color: #4CAF50;");
        m_reconnectBtn->setEnabled(!m_reconnectOnCooldown);
        m_trayIcon->setLoggedIn(true);
    } else {
        m_statusLabel->setText("未登录");
        m_statusLabel->setStyleSheet("font-weight: bold; color: #f44336;");
        m_reconnectBtn->setEnabled(false);
        m_trayIcon->setLoggedIn(false);
    }

    QString ip = m_ipDetector->detectLocalIp();
    m_ipLabel->setText(ip.isEmpty() ? "Unknown" : ip);
}

void MainWindow::performLogin(const QString& username, const QString& password,
                               const QString& operator_, bool isReconnect)
{
    QString localIp = m_ipDetector->detectLocalIp();
    QString gateway = m_networkManager->currentGateway();

    if (isReconnect) {
        m_reconnectOnCooldown = true;
        m_reconnectCooldown->start();
        m_reconnectBtn->setEnabled(false);
    }

    m_loginManager->login(username, password, operator_, localIp, gateway);
}

void MainWindow::onLoginClicked()
{
    QString username = m_usernameEdit->text().trimmed();
    QString password = m_passwordEdit->text();
    QString operator_ = m_operatorCombo->currentData().toString();

    if (username.isEmpty() || password.isEmpty()) {
        QMessageBox::warning(this, "输入错误", "用户名和密码不能为空。");
        return;
    }

    if (m_networkManager->currentGateway().isEmpty()) {
        QMessageBox::warning(this, "网关错误",
                             "网关尚未就绪，请稍候或在高级设置中手动设置网关地址。");
        return;
    }

    m_statusLabel->setText("连接中...");
    m_statusLabel->setStyleSheet("font-weight: bold; color: #FF9800;");

    // Sync to config and persist immediately
    m_currentConfig->username = username;
    m_currentConfig->operator_ = operator_;
    if (m_currentConfig->rememberPassword) {
        m_currentConfig->password = password;
    }
    // Always keep an in-session copy so reconnect works even without "remember password".
    m_sessionPassword = password;
    saveSettings();

    performLogin(username, password, operator_);
}

void MainWindow::onReconnectClicked()
{
    if (m_reconnectOnCooldown)
        return;

    QString username = m_currentConfig->username;
    QString password = m_currentConfig->password;
    QString oper = m_currentConfig->operator_;

    // Fall back to in-session password when "remember password" was off.
    if (password.isEmpty())
        password = m_sessionPassword;

    if (username.isEmpty() || password.isEmpty()) {
        QMessageBox::warning(this, "重连错误",
                             "没有已保存的凭据，请先手动登录一次。");
        return;
    }

    m_statusLabel->setText("重连中...");
    m_statusLabel->setStyleSheet("font-weight: bold; color: #FF9800;");

    Logger::instance().info("Manual reconnect triggered");
    performLogin(username, password, oper, true);
}

void MainWindow::onRefreshIp()
{
    QString ip = m_ipDetector->detectLocalIp();
    m_ipLabel->setText(ip.isEmpty() ? "Unknown" : ip);
    Logger::instance().info(QString("IP refreshed: %1").arg(ip));
}

void MainWindow::onRefreshNetwork()
{
    Logger::instance().info("Manual network refresh triggered");
    flushDns();
    flushArp();
    onRefreshIp();

    if (m_currentConfig->customGateway.isEmpty()) {
        m_networkManager->probeGateway();
    }

    if (m_currentConfig->heartbeatEnabled) {
        m_heartbeat->stop();
        m_heartbeat->start(m_currentConfig->heartbeatInterval);
    }

    m_connectionLabel->setText("刷新中...");
    m_connectionLabel->setStyleSheet("color: #FF9800;");
}

void MainWindow::onAdvancedSettings()
{
    m_advancedDialog->loadFromConfig(*m_currentConfig);
    if (m_advancedDialog->exec() == QDialog::Accepted) {
        m_advancedDialog->saveToConfig(*m_currentConfig);
        m_heartbeat->setInterval(m_currentConfig->heartbeatInterval);
        m_heartbeat->setCheckUrls(m_currentConfig->pingUrls);
        m_heartbeatCheck->setChecked(m_currentConfig->heartbeatEnabled);
        if (m_currentConfig->heartbeatEnabled)
            m_heartbeat->start(m_currentConfig->heartbeatInterval);
        m_autoStartManager->setAutoStart(m_currentConfig->autoStart);

        if (!m_currentConfig->customGateway.isEmpty()) {
            m_networkManager->setCustomGateway(m_currentConfig->customGateway);
        } else {
            m_networkManager->clearCustomGateway();
        }

        m_configManager->saveConfig(*m_currentConfig);
        Logger::instance().info("Advanced settings applied");
    }
}

void MainWindow::onLoginSuccess()
{
    m_currentConfig->lastLoginTime = QDateTime::currentDateTime();
    m_lastLoginLabel->setText(m_currentConfig->lastLoginTime.toString("yyyy-MM-dd hh:mm:ss"));

    saveSettings();

    updateStatusDisplay();
    m_warmupCount = 0;
    m_warmupTimer->start(1000);

    if (m_currentConfig->heartbeatEnabled && !m_heartbeat->isRunning()) {
        m_heartbeat->start(m_currentConfig->heartbeatInterval);
    }
}

void MainWindow::onLoginFailed(const QString& error)
{
    updateStatusDisplay();
    QMessageBox::critical(this, "登录失败", error);
}

void MainWindow::onConnectionLost()
{
    m_connectionLabel->setText("已断开");
    m_connectionLabel->setStyleSheet("color: #f44336;");
    m_trayIcon->setConnectionStatus(false);
    Logger::instance().info("Connection lost");
}

void MainWindow::onConnectionRestored()
{
    m_connectionLabel->setText("已连接");
    m_connectionLabel->setStyleSheet("color: #4CAF50;");
    m_trayIcon->setConnectionStatus(true);
    updateStatusDisplay();
}

void MainWindow::onConnectionAlive()
{
    m_connectionLabel->setText("已连接");
    m_connectionLabel->setStyleSheet("color: #4CAF50;");
    m_trayIcon->setConnectionStatus(true);
}

void MainWindow::onTrayActivated(QSystemTrayIcon::ActivationReason reason)
{
    if (reason == QSystemTrayIcon::DoubleClick) {
        show();
        raise();
        activateWindow();
    }
}

void MainWindow::onAutoLoginTriggered()
{
    if (!m_currentConfig->autoLogin || m_currentConfig->username.isEmpty())
        return;

    if (m_networkManager->currentGateway().isEmpty()) {
        m_autoLoginRetryCount++;
        if (m_autoLoginRetryCount < 15) {
            Logger::instance().info(QString("Auto-login: gateway not ready, retry %1/15")
                                        .arg(m_autoLoginRetryCount));
            m_autoLoginTimer->start(3000);
        } else {
            Logger::instance().error("Auto-login: gateway not ready after 15 retries, giving up");
            m_autoLoginRetryCount = 0;
        }
        return;
    }

    m_autoLoginRetryCount = 0;
    Logger::instance().info("Performing auto-login...");
    performLogin(m_currentConfig->username, m_currentConfig->password,
                  m_currentConfig->operator_);

    // Keep an in-session copy for possible reconnect later.
    m_sessionPassword = m_currentConfig->password;
}

void MainWindow::onGatewayReady(const QString& gateway)
{
    m_gatewayLabel->setText(gateway);
    m_gatewayLabel->setStyleSheet("color: #4CAF50;");
    Logger::instance().info(QString("Gateway ready: %1").arg(gateway));
}

void MainWindow::onGatewayProbeFailed()
{
    m_gatewayLabel->setText("探测失败");
    m_gatewayLabel->setStyleSheet("color: #f44336;");
}

void MainWindow::onGatewayChanged(const QString& gateway)
{
    m_gatewayLabel->setText(gateway);
}

void MainWindow::onNetworkTypeChanged(NetworkType oldType, NetworkType newType)
{
    Logger::instance().info(QString("Network type changed: %1 -> %2")
                                .arg(oldType == NetworkType::Wired ? "wired" : "wireless")
                                .arg(newType == NetworkType::Wired ? "wired" : "wireless"));
    onRefreshIp();
}

void MainWindow::onNewLogEntry(const LogEntry& entry)
{
    int filterLevel = m_logLevelFilter->currentData().toInt();
    if (filterLevel != -1 && static_cast<int>(entry.level) < filterLevel)
        return;

    QString timeStr = entry.timestamp.toString("hh:mm:ss");
    QString levelStr;
    QString color;

    switch (entry.level) {
    case LogLevel::Debug:   levelStr = "DEBUG"; color = "#888888"; break;
    case LogLevel::Info:    levelStr = "INFO "; color = "#2196F3"; break;
    case LogLevel::Warning: levelStr = "WARN "; color = "#FF9800"; break;
    case LogLevel::Error:   levelStr = "ERROR"; color = "#f44336"; break;
    }

    QString html = QString("<span style='color:%1'>[%2] %3 %4</span>")
                       .arg(color, timeStr, levelStr, entry.message.toHtmlEscaped());

    m_logPanel->append(html);

    QScrollBar* sb = m_logPanel->verticalScrollBar();
    sb->setValue(sb->maximum());
}

void MainWindow::onReconnectCooldownTimeout()
{
    m_reconnectOnCooldown = false;
    if (m_loginManager->isLoggedIn()) {
        m_reconnectBtn->setEnabled(true);
    }
}

void MainWindow::closeEvent(QCloseEvent* event)
{
    hide();
    event->ignore();
}

void MainWindow::flushDns()
{
    QProcess* process = new QProcess(this);
    connect(process, QOverload<int, QProcess::ExitStatus>::of(&QProcess::finished),
            this, [this, process](int exitCode, QProcess::ExitStatus) {
        if (exitCode == 0) {
            Logger::instance().info("DNS cache flushed successfully");
        } else {
            Logger::instance().warning("Failed to flush DNS cache");
        }
        process->deleteLater();
    });

    process->start("ipconfig", QStringList() << "/flushdns");
    Logger::instance().info("Flushing DNS cache...");
}

void MainWindow::flushArp()
{
    QProcess* process = new QProcess(this);
    connect(process, QOverload<int, QProcess::ExitStatus>::of(&QProcess::finished),
            this, [this, process](int exitCode, QProcess::ExitStatus) {
        if (exitCode == 0) {
            Logger::instance().debug("ARP cache cleared");
        }
        process->deleteLater();
    });

    process->start("arp", QStringList() << "-d" << "*");
    Logger::instance().debug("Clearing ARP cache...");
}