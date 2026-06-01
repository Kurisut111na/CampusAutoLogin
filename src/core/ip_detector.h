#pragma once

#include <QObject>
#include <QString>
#include <QHostAddress>

enum class NetworkType {
    Unknown,
    Wired,
    Wireless
};

class IpDetector : public QObject
{
    Q_OBJECT

public:
    explicit IpDetector(QObject* parent = nullptr);

    QString detectLocalIp() const;
    NetworkType detectNetworkType() const;

    static bool isPrivateIPv4(const QHostAddress& addr);
    static bool isCampusNetwork(const QHostAddress& addr);
    static bool isVirtualAdapter(const QString& interfaceName, const QString& description);

signals:
    void networkTypeChanged(NetworkType oldType, NetworkType newType);
};