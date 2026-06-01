#include "config_manager.h"
#include "crypto_utils.h"
#include "logger.h"

#include <QCoreApplication>
#include <QDir>
#include <QJsonDocument>
#include <QJsonObject>
#include <QJsonArray>
#include <QFile>
#include <QStandardPaths>

ConfigManager::ConfigManager(QObject* parent)
    : QObject(parent)
{
    m_configDir = QStandardPaths::writableLocation(QStandardPaths::AppLocalDataLocation);
    QDir().mkpath(m_configDir);
    m_configFile = m_configDir + "/config.json";

    Logger::instance().info(QString("Config file path: %1").arg(m_configFile));
}

QString ConfigManager::configFilePath() const
{
    return m_configFile;
}

bool ConfigManager::loadConfig(AppConfig& config)
{
    QFile file(m_configFile);
    if (!file.exists()) {
        Logger::instance().info("Config file not found, using defaults");
        return false;
    }

    if (!file.open(QIODevice::ReadOnly | QIODevice::Text)) {
        QString err = QString("Cannot open config file: %1").arg(file.errorString());
        Logger::instance().error(err);
        emit configError(err);
        return false;
    }

    QByteArray data = file.readAll();
    file.close();

    QJsonParseError parseError;
    QJsonDocument doc = QJsonDocument::fromJson(data, &parseError);
    if (parseError.error != QJsonParseError::NoError) {
        QString err = QString("Config JSON parse error: %1").arg(parseError.errorString());
        Logger::instance().error(err);
        emit configError(err);
        return false;
    }

    QJsonObject root = doc.object();

    config.username = root.value("username").toString();

    // Password is stored DPAPI-encrypted + base64.
    // If decryption fails the data is either old-format (lost AES key) or
    // plaintext from a broken intermediate build — use it as-is and it will
    // be re-encrypted on the next save.
    QString rawPassword = root.value("password").toString();
    if (!rawPassword.isEmpty()) {
        QString decrypted = CryptoUtils::unprotect(rawPassword);
        if (!decrypted.isEmpty()) {
            config.password = decrypted;
        } else {
            // Could be old-format or plaintext.  Accept as-is; next save
            // will re-encrypt with DPAPI.
            config.password = rawPassword;
            Logger::instance().warning(
                "Password decryption failed — using raw value (will re-encrypt on save)");
        }
    } else {
        config.password.clear();
    }

    config.operator_ = root.value("operator").toString();
    config.autoLogin = root.value("auto_login").toBool(false);
    config.rememberPassword = root.value("remember_password").toBool(true);
    config.autoStart = root.value("auto_start").toBool(false);
    config.startMinimized = root.value("start_minimized").toBool(false);
    config.heartbeatEnabled = root.value("heartbeat_enabled").toBool(true);
    config.heartbeatInterval = root.value("heartbeat_interval").toInt(45);
    config.customGateway = root.value("custom_gateway").toString();

    QJsonArray urls = root.value("ping_urls").toArray();
    if (!urls.isEmpty()) {
        for (const auto& v : urls)
            config.pingUrls.append(v.toString());
    } else {
        config.pingUrls = QStringList{
            "https://www.baidu.com",
            "https://www.bing.com"
        };
    }

    QString lastLoginStr = root.value("last_login_time").toString();
    if (!lastLoginStr.isEmpty()) {
        config.lastLoginTime = QDateTime::fromString(lastLoginStr, "yyyy-MM-dd hh:mm:ss");
    }

    Logger::instance().info("Config loaded successfully");
    emit configLoaded(config);
    return true;
}

bool ConfigManager::saveConfig(const AppConfig& config)
{
    QJsonObject root;

    root["username"] = config.username;

    // Encrypt password with DPAPI before persisting.
    if (!config.password.isEmpty()) {
        QString encrypted = CryptoUtils::protect(config.password);
        if (!encrypted.isEmpty()) {
            root["password"] = encrypted;
        } else {
            // DPAPI failed — fall back to storing empty so we don't write
            // plaintext to disk.
            Logger::instance().error("DPAPI protect failed — password NOT saved to disk");
            root["password"] = QString();
        }
    } else {
        root["password"] = QString();
    }

    root["operator"] = config.operator_;
    root["auto_login"] = config.autoLogin;
    root["remember_password"] = config.rememberPassword;
    root["auto_start"] = config.autoStart;
    root["start_minimized"] = config.startMinimized;
    root["heartbeat_enabled"] = config.heartbeatEnabled;
    root["heartbeat_interval"] = config.heartbeatInterval;
    root["custom_gateway"] = config.customGateway;

    QJsonArray urls;
    for (const auto& url : config.pingUrls)
        urls.append(url);
    root["ping_urls"] = urls;

    if (config.lastLoginTime.isValid()) {
        root["last_login_time"] = config.lastLoginTime.toString("yyyy-MM-dd hh:mm:ss");
    } else {
        root["last_login_time"] = QString();
    }

    QJsonDocument doc(root);
    QFile file(m_configFile);
    if (!file.open(QIODevice::WriteOnly | QIODevice::Text | QIODevice::Truncate)) {
        QString err = QString("Cannot write config file: %1").arg(file.errorString());
        Logger::instance().error(err);
        emit configError(err);
        return false;
    }

    file.write(doc.toJson(QJsonDocument::Indented));
    file.close();

    Logger::instance().info("Config saved successfully");
    emit configSaved();
    return true;
}