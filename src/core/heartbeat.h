#pragma once

#include <QObject>
#include <QTimer>
#include <QNetworkAccessManager>
#include <QNetworkReply>
#include <QStringList>

class Heartbeat : public QObject
{
    Q_OBJECT

public:
    explicit Heartbeat(QObject* parent = nullptr);

    void start(int intervalSeconds = 45);
    void stop();
    bool isRunning() const;
    bool isConnected() const;

    void setInterval(int seconds);
    void setCheckUrls(const QStringList& urls);

signals:
    void connectionAlive();
    void connectionLost();
    void connectionRestored();
    void reconnectRequested();

private slots:
    void onCheckTimeout();
    void onHeadReplyFinished(QNetworkReply* reply);
    void onRetryTimeout();

private:
    void sendHeadRequest(const QString& url);
    void startRetrySequence();
    void resetRetryState();
    void cancelActiveRequest();

    QTimer* m_checkTimer;
    QTimer* m_retryTimer;
    QNetworkAccessManager* m_networkManager;
    QNetworkReply* m_activeReply = nullptr;

    bool m_running = false;
    bool m_wasConnected = true;
    int m_consecutiveFailures = 0;

    static constexpr int FAILURE_THRESHOLD = 2;

    bool m_retrying = false;
    int m_retryCount = 0;
    int m_retryDelay = 0;

    static constexpr int MAX_RETRIES = 5;
    static const QList<int> RETRY_DELAYS;

    QStringList m_checkUrls;
    int m_currentUrlIndex = 0;

    static constexpr int REQUEST_TIMEOUT = 5000;
};