#pragma once

#include <QObject>
#include <QString>
#include <QNetworkAccessManager>
#include <QNetworkReply>
#include <QTimer>

#include "ip_detector.h"

class NetworkManager : public QObject
{
    Q_OBJECT

public:
    explicit NetworkManager(IpDetector* ipDetector, QObject* parent = nullptr);

    QString currentGateway() const;
    QString probeGateway();

    void setCustomGateway(const QString& gateway);
    void clearCustomGateway();

signals:
    void gatewayReady(const QString& gateway);
    void gatewayProbeFailed();
    void gatewayChanged(const QString& newGateway);

private slots:
    void onNetworkTypeCheck();
    void onProbeReplyFinished(QNetworkReply* reply);
    void onProbeTimeout();

private:
    void startProbe(const QString& gateway);
    QString wiredGateway() const;
    QString wirelessGateway() const;

    IpDetector* m_ipDetector;
    QNetworkAccessManager* m_networkManager;
    QTimer* m_networkTypeCheckTimer;
    QTimer* m_probeTimeout;

    QString m_currentGateway;
    QString m_customGateway;
    NetworkType m_lastNetworkType = NetworkType::Unknown;

    QNetworkReply* m_activeProbe = nullptr;
    QStringList m_probeCandidates;
    int m_probeIndex = 0;
    bool m_probing = false;
};