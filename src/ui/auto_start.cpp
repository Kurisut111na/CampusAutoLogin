#include "auto_start.h"
#include "logger.h"

#include <QCoreApplication>
#include <QDir>
#include <QSettings>

AutoStartManager::AutoStartManager(QObject* parent)
    : QObject(parent)
{
    m_enabled = isAutoStartEnabled();
}

bool AutoStartManager::isAutoStartEnabled() const
{
    QSettings settings(registryPath(), QSettings::NativeFormat);
    return settings.contains("CampusAutoLogin");
}

bool AutoStartManager::setAutoStart(bool enabled)
{
    QSettings settings(registryPath(), QSettings::NativeFormat);

    if (enabled) {
        QString appPath = QDir::toNativeSeparators(applicationPath());
        settings.setValue("CampusAutoLogin", appPath);
        Logger::instance().info(QString("Auto-start enabled: %1").arg(appPath));
    } else {
        settings.remove("CampusAutoLogin");
        Logger::instance().info("Auto-start disabled");
    }

    m_enabled = enabled;
    emit autoStartChanged(enabled);
    return true;
}

QString AutoStartManager::registryPath() const
{
    return "HKEY_CURRENT_USER\\Software\\Microsoft\\Windows\\CurrentVersion\\Run";
}

QString AutoStartManager::applicationPath() const
{
    return QCoreApplication::applicationFilePath();
}