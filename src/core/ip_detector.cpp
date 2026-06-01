#include "ip_detector.h"
#include "logger.h"

#include <QNetworkInterface>
#include <QList>

static const QStringList VIRTUAL_KEYWORDS = {
    "virtualbox", "vmware", "hyper-v", "wsl",
    "virtual", "vbox", "vpn", "tunnel", "loopback",
    "bluetooth", "pseudo"
};

IpDetector::IpDetector(QObject* parent)
    : QObject(parent)
{
}

QString IpDetector::detectLocalIp() const
{
    const QList<QNetworkInterface> interfaces = QNetworkInterface::allInterfaces();

    struct Candidate {
        QString ip;
        int priority;
    };

    QList<Candidate> candidates;

    for (const QNetworkInterface& iface : interfaces) {
        if (iface.flags().testFlag(QNetworkInterface::IsLoopBack))
            continue;
        if (!iface.flags().testFlag(QNetworkInterface::IsUp))
            continue;
        if (!iface.flags().testFlag(QNetworkInterface::IsRunning))
            continue;

        QString name = iface.humanReadableName().toLower();
        QString hwName = iface.name().toLower();

        if (isVirtualAdapter(hwName, name))
            continue;

        const QList<QNetworkAddressEntry> entries = iface.addressEntries();
        for (const QNetworkAddressEntry& entry : entries) {
            QHostAddress ip = entry.ip();
            if (ip.protocol() != QAbstractSocket::IPv4Protocol)
                continue;
            if (ip.isLoopback())
                continue;

            int priority = 0;

            if (isCampusNetwork(ip)) {
                priority += 100;
            } else if (isPrivateIPv4(ip)) {
                priority += 50;
            } else {
                continue;
            }

            bool isWireless = name.contains("wireless")
                           || name.contains("wi-fi")
                           || name.contains("wlan")
                           || hwName.contains("wlan");
            if (!isWireless) {
                priority += 20;
            }

            candidates.append({ip.toString(), priority});
        }
    }

    if (candidates.isEmpty()) {
        Logger::instance().warning("No valid IPv4 address detected");
        return {};
    }

    std::sort(candidates.begin(), candidates.end(),
              [](const Candidate& a, const Candidate& b) {
                  return a.priority > b.priority;
              });

    QString selected = candidates.first().ip;
    Logger::instance().info(QString("Detected local IP: %1 (priority: %2)")
                                .arg(selected)
                                .arg(candidates.first().priority));
    return selected;
}

NetworkType IpDetector::detectNetworkType() const
{
    const QList<QNetworkInterface> interfaces = QNetworkInterface::allInterfaces();

    bool hasWired = false;
    bool hasWireless = false;

    for (const QNetworkInterface& iface : interfaces) {
        if (iface.flags().testFlag(QNetworkInterface::IsLoopBack))
            continue;
        if (!iface.flags().testFlag(QNetworkInterface::IsUp))
            continue;
        if (!iface.flags().testFlag(QNetworkInterface::IsRunning))
            continue;

        QString name = iface.humanReadableName().toLower();
        QString hwName = iface.name().toLower();

        if (isVirtualAdapter(hwName, name))
            continue;

        const QList<QNetworkAddressEntry> entries = iface.addressEntries();
        bool hasValidIp = false;
        for (const QNetworkAddressEntry& entry : entries) {
            QHostAddress ip = entry.ip();
            if (ip.protocol() == QAbstractSocket::IPv4Protocol
                && !ip.isLoopback()
                && isPrivateIPv4(ip)) {
                hasValidIp = true;
                break;
            }
        }
        if (!hasValidIp)
            continue;

        bool isWireless = name.contains("wireless")
                       || name.contains("wi-fi")
                       || name.contains("wlan")
                       || hwName.contains("wlan");

        if (isWireless) {
            hasWireless = true;
        } else {
            hasWired = true;
        }
    }

    if (hasWired) {
        Logger::instance().info("Network type: Wired (Ethernet)");
        return NetworkType::Wired;
    }
    if (hasWireless) {
        Logger::instance().info("Network type: Wireless (Wi-Fi)");
        return NetworkType::Wireless;
    }

    Logger::instance().warning("Network type: Unknown");
    return NetworkType::Unknown;
}

bool IpDetector::isPrivateIPv4(const QHostAddress& addr)
{
    if (addr.protocol() != QAbstractSocket::IPv4Protocol)
        return false;

    quint32 ipv4 = addr.toIPv4Address();

    quint32 a = (ipv4 >> 24) & 0xFF;
    quint32 b = (ipv4 >> 16) & 0xFF;

    if (a == 10)
        return true;

    if (a == 172 && b >= 16 && b <= 31)
        return true;

    if (a == 192 && b == 168)
        return true;

    return false;
}

bool IpDetector::isCampusNetwork(const QHostAddress& addr)
{
    if (addr.protocol() != QAbstractSocket::IPv4Protocol)
        return false;

    quint32 ipv4 = addr.toIPv4Address();
    quint32 a = (ipv4 >> 24) & 0xFF;

    return (a == 10);
}

bool IpDetector::isVirtualAdapter(const QString& interfaceName,
                                   const QString& description)
{
    QString combined = (interfaceName + " " + description).toLower();
    for (const QString& keyword : VIRTUAL_KEYWORDS) {
        if (combined.contains(keyword))
            return true;
    }
    return false;
}