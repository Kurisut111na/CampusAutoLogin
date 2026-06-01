#pragma once

#include <QObject>
#include <QString>
#include <QDateTime>

struct AppConfig {
    QString username;
    QString password;
    QString operator_;
    bool autoLogin = false;
    bool rememberPassword = true;
    bool autoStart = false;
    bool startMinimized = false;
    bool heartbeatEnabled = true;
    int heartbeatInterval = 45;
    QString customGateway;
    QStringList pingUrls;
    QDateTime lastLoginTime;
};

class ConfigManager : public QObject
{
    Q_OBJECT

public:
    explicit ConfigManager(QObject* parent = nullptr);

    bool loadConfig(AppConfig& config);
    bool saveConfig(const AppConfig& config);

    QString configFilePath() const;

signals:
    void configSaved();
    void configLoaded(const AppConfig& config);
    void configError(const QString& error);

private:
    QString m_configDir;
    QString m_configFile;
};