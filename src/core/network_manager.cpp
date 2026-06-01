#include "network_manager.h"
#include "logger.h"

#include <QNetworkRequest>

NetworkManager::NetworkManager(IpDetector* ipDetector, QObject* parent)
    : QObject(parent)
    , m_ipDetector(ipDetector)
    , m_networkManager(new QNetworkAccessManager(this))
    , m_networkTypeCheckTimer(new QTimer(this))
    , m_probeTimeout(new QTimer(this))
{
    m_probeTimeout->setSingleShot(true);

    connect(m_networkTypeCheckTimer, &QTimer::timeout,
            this, &NetworkManager::onNetworkTypeCheck);
    connect(m_networkManager, &QNetworkAccessManager::finished,
            this, &NetworkManager::onProbeReplyFinished);
    connect(m_probeTimeout, &QTimer::timeout,
            this, &NetworkManager::onProbeTimeout);

    m_networkTypeCheckTimer->start(10000);
}

QString NetworkManager::currentGateway() const
{
    if (!m_customGateway.isEmpty())
        return m_customGateway;
    return m_currentGateway;
}

QString NetworkManager::probeGateway()
{
    if (!m_customGateway.isEmpty()) {
        m_currentGateway = m_customGateway;
        Logger::instance().info(QString("Using custom gateway: %1").arg(m_currentGateway));
        emit gatewayReady(m_currentGateway);
        return m_currentGateway;
    }

    if (m_probing) {
        Logger::instance().debug("Gateway probe already in progress");
        return {};
    }

    m_probing = true;
    m_probeCandidates.clear();
    m_probeIndex = 0;

    NetworkType type = m_ipDetector->detectNetworkType();
    m_lastNetworkType = type;

    if (type == NetworkType::Wired) {
        m_probeCandidates.append(wiredGateway());
        m_probeCandidates.append(wirelessGateway());
    } else {
        m_probeCandidates.append(wirelessGateway());
        m_probeCandidates.append(wiredGateway());
    }

    Logger::instance().info(QString("Gateway probe: type=%1, candidates=%2")
                                .arg(type == NetworkType::Wired ? "wired" : "wireless")
                                .arg(m_probeCandidates.join(", ")));

    startProbe(m_probeCandidates.first());
    return {};
}

void NetworkManager::setCustomGateway(const QString& gateway)
{
    m_customGateway = gateway;
    m_currentGateway = gateway;
    Logger::instance().info(QString("Custom gateway set: %1").arg(gateway));
    emit gatewayChanged(gateway);
    emit gatewayReady(gateway);
}

void NetworkManager::clearCustomGateway()
{
    m_customGateway.clear();
    Logger::instance().info("Custom gateway cleared, will auto-probe");
    probeGateway();
}

void NetworkManager::onNetworkTypeCheck()
{
    NetworkType currentType = m_ipDetector->detectNetworkType();
    if (currentType != m_lastNetworkType && currentType != NetworkType::Unknown) {
        Logger::instance().info(QString("Network type changed: %1 -> %2")
                                    .arg(m_lastNetworkType == NetworkType::Wired ? "wired" : "wireless")
                                    .arg(currentType == NetworkType::Wired ? "wired" : "wireless"));
        emit m_ipDetector->networkTypeChanged(m_lastNetworkType, currentType);
        m_lastNetworkType = currentType;

        if (m_customGateway.isEmpty()) {
            probeGateway();
        }
    }
}

void NetworkManager::startProbe(const QString& gateway)
{
    if (m_activeProbe) {
        m_activeProbe->abort();
        m_activeProbe->deleteLater();
        m_activeProbe = nullptr;
    }

    QString probeUrl = QString("http://%1/").arg(gateway);
    Logger::instance().info(QString("Probing gateway: %1").arg(probeUrl));

    QNetworkRequest request{QUrl(probeUrl)};
    request.setTransferTimeout(5000);

    m_activeProbe = m_networkManager->head(request);
    m_probeTimeout->start(5000);
}

void NetworkManager::onProbeReplyFinished(QNetworkReply* reply)
{
    if (reply != m_activeProbe)
        return;

    m_activeProbe = nullptr;
    m_probeTimeout->stop();

    QString probedGateway = m_probeCandidates.value(m_probeIndex);

    if (reply->error() == QNetworkReply::NoError
        || reply->error() == QNetworkReply::ContentNotFoundError
        || reply->error() == QNetworkReply::ContentAccessDenied) {
        m_currentGateway = probedGateway;
        m_probing = false;
        Logger::instance().info(QString("Gateway probe success: %1").arg(m_currentGateway));
        emit gatewayReady(m_currentGateway);
        emit gatewayChanged(m_currentGateway);
        reply->deleteLater();
        return;
    }

    Logger::instance().warning(QString("Gateway probe failed for %1: %2")
                                   .arg(probedGateway, reply->errorString()));
    reply->deleteLater();

    m_probeIndex++;
    if (m_probeIndex < m_probeCandidates.size()) {
        startProbe(m_probeCandidates.at(m_probeIndex));
    } else {
        m_probing = false;
        Logger::instance().error("All gateway probes failed");
        emit gatewayProbeFailed();
    }
}

void NetworkManager::onProbeTimeout()
{
    if (m_activeProbe) {
        m_activeProbe->abort();
        m_activeProbe->deleteLater();
        m_activeProbe = nullptr;
    }

    QString probedGateway = m_probeCandidates.value(m_probeIndex);
    Logger::instance().warning(QString("Gateway probe timeout for %1").arg(probedGateway));

    m_probeIndex++;
    if (m_probeIndex < m_probeCandidates.size()) {
        startProbe(m_probeCandidates.at(m_probeIndex));
    } else {
        m_probing = false;
        Logger::instance().error("All gateway probes timed out");
        emit gatewayProbeFailed();
    }
}

QString NetworkManager::wiredGateway() const
{
    return "10.0.1.5";
}

QString NetworkManager::wirelessGateway() const
{
    return "1.2.3.4";
}