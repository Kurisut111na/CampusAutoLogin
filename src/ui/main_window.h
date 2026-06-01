#pragma once

#include <QMainWindow>
#include <QSystemTrayIcon>
#include <QLabel>
#include <QPushButton>
#include <QComboBox>
#include <QLineEdit>
#include <QCheckBox>
#include <QGroupBox>
#include <QTimer>
#include <QTextEdit>

#include "ip_detector.h"
#include "logger.h"

class ConfigManager;
class IpDetector;
class LoginManager;
class Heartbeat;
class NetworkManager;
class TrayIcon;
class AutoStartManager;
class AdvancedSettingsDialog;
struct AppConfig;

class MainWindow : public QMainWindow
{
    Q_OBJECT

public:
    explicit MainWindow(QWidget* parent = nullptr);
    ~MainWindow();

    bool shouldStartMinimized() const;

protected:
    void closeEvent(QCloseEvent* event) override;

private slots:
    void onLoginClicked();
    void onReconnectClicked();
    void onRefreshIp();
    void onRefreshNetwork();
    void onAdvancedSettings();
    void onLoginSuccess();
    void onLoginFailed(const QString& error);
    void onConnectionLost();
    void onConnectionRestored();
    void onConnectionAlive();
    void onTrayActivated(QSystemTrayIcon::ActivationReason reason);
    void onAutoLoginTriggered();
    void onGatewayReady(const QString& gateway);
    void onGatewayProbeFailed();
    void onGatewayChanged(const QString& gateway);
    void onNetworkTypeChanged(NetworkType oldType, NetworkType newType);
    void onNewLogEntry(const LogEntry& entry);
    void onReconnectCooldownTimeout();

private:
    void setupUi();
    void setupConnections();
    void loadSettings();
    void saveSettings();
    void updateStatusDisplay();
    void flushDns();
    void flushArp();
    void performLogin(const QString& username, const QString& password,
                      const QString& operator_, bool isReconnect = false);
    void appendLog(const QString& message, const QString& color = "#000000");

    ConfigManager*        m_configManager;
    IpDetector*           m_ipDetector;
    LoginManager*         m_loginManager;
    Heartbeat*            m_heartbeat;
    NetworkManager*       m_networkManager;
    TrayIcon*             m_trayIcon;
    AutoStartManager*     m_autoStartManager;
    AdvancedSettingsDialog* m_advancedDialog;
    AppConfig*            m_currentConfig;

    QLineEdit*   m_usernameEdit;
    QLineEdit*   m_passwordEdit;
    QPushButton* m_showPasswordBtn;
    QComboBox*   m_operatorCombo;
    QCheckBox*   m_autoLoginCheck;
    QCheckBox*   m_rememberPasswordCheck;
    QCheckBox*   m_heartbeatCheck;

    QLabel*      m_statusLabel;
    QLabel*      m_ipLabel;
    QLabel*      m_gatewayLabel;
    QLabel*      m_lastLoginLabel;
    QLabel*      m_connectionLabel;

    QPushButton* m_loginBtn;
    QPushButton* m_reconnectBtn;
    QPushButton* m_refreshIpBtn;
    QPushButton* m_refreshNetworkBtn;
    QPushButton* m_advancedBtn;

    QTextEdit*   m_logPanel;
    QPushButton* m_clearChromeDnsBtn;
    QPushButton* m_clearLogBtn;
    QComboBox*   m_logLevelFilter;

    QTimer*      m_autoLoginTimer;
    QTimer*      m_reconnectCooldown;
    QTimer*      m_dnsDelayTimer;
    QTimer*      m_warmupTimer;
    QTimer*      m_saveTimer;
    int           m_warmupCount = 0;

    bool m_passwordVisible = false;
    bool m_reconnectOnCooldown = false;
    bool m_loading = false;
    int m_autoLoginRetryCount = 0;

    // Only valid for the current session — lets reconnect work even when
    // "remember password" is off.  Never persisted.
    QString m_sessionPassword;
};