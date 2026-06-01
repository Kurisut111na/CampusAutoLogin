#pragma once

#include <QObject>

class AutoStartManager : public QObject
{
    Q_OBJECT

public:
    explicit AutoStartManager(QObject* parent = nullptr);

    bool isAutoStartEnabled() const;
    bool setAutoStart(bool enabled);

signals:
    void autoStartChanged(bool enabled);
    void autoStartError(const QString& error);

private:
    QString registryPath() const;
    QString applicationPath() const;

    bool m_enabled = false;
};