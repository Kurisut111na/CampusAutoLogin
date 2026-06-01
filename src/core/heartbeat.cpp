#include "heartbeat.h"
#include "logger.h"

#include <QNetworkRequest>

const QList<int> Heartbeat::RETRY_DELAYS = {1000, 2000, 4000, 8000, 16000};

Heartbeat::Heartbeat(QObject* parent)
    : QObject(parent)
    , m_checkTimer(new QTimer(this))
    , m_retryTimer(new QTimer(this))
    , m_networkManager(new QNetworkAccessManager(this))
{
    m_checkUrls << "https://www.baidu.com"
                << "https://www.bing.com";

    m_retryTimer->setSingleShot(true);

    connect(m_checkTimer, &QTimer::timeout, this, &Heartbeat::onCheckTimeout);
    connect(m_retryTimer, &QTimer::timeout, this, &Heartbeat::onRetryTimeout);
    connect(m_networkManager, &QNetworkAccessManager::finished,
            this, &Heartbeat::onHeadReplyFinished);
}

void Heartbeat::start(int intervalSeconds)
{
    if (m_running)
        return;

    m_running = true;
    m_wasConnected = true;
    m_consecutiveFailures = 0;

    // Fire first check after 1 second, then use full interval
    m_checkTimer->setInterval(intervalSeconds * 1000);
    QTimer::singleShot(1000, this, [this]() {
        if (m_running && !m_retrying && !m_activeReply) {
            onCheckTimeout();
            if (m_running) {
                m_checkTimer->start();
            }
        }
    });
    Logger::instance().info(QString("Heartbeat started, interval: %1s").arg(intervalSeconds));
}

void Heartbeat::stop()
{
    if (!m_running)
        return;

    m_running = false;
    m_checkTimer->stop();
    m_retryTimer->stop();
    m_retrying = false;
    cancelActiveRequest();

    Logger::instance().info("Heartbeat stopped");
}

bool Heartbeat::isRunning() const
{
    return m_running;
}

bool Heartbeat::isConnected() const
{
    return m_wasConnected;
}

void Heartbeat::setInterval(int seconds)
{
    if (m_running) {
        m_checkTimer->start(seconds * 1000);
    }
    Logger::instance().info(QString("Heartbeat interval changed to: %1s").arg(seconds));
}

void Heartbeat::setCheckUrls(const QStringList& urls)
{
    if (!urls.isEmpty()) {
        m_checkUrls = urls;
        m_currentUrlIndex = 0;
        Logger::instance().info(QString("Heartbeat URLs updated: %1 urls").arg(urls.size()));
    }
}

void Heartbeat::onCheckTimeout()
{
    if (m_retrying || m_activeReply)
        return;

    QString url = m_checkUrls.at(m_currentUrlIndex);
    m_currentUrlIndex = (m_currentUrlIndex + 1) % m_checkUrls.size();

    Logger::instance().debug(QString("Heartbeat: sending HEAD to %1").arg(url));
    sendHeadRequest(url);
}

void Heartbeat::onHeadReplyFinished(QNetworkReply* reply)
{
    if (reply != m_activeReply)
        return;

    m_activeReply = nullptr;

    bool isAlive = (reply->error() == QNetworkReply::NoError);

    if (isAlive) {
        m_consecutiveFailures = 0;

        if (m_retrying) {
            m_retrying = false;
            m_retryCount = 0;
            m_retryTimer->stop();
            Logger::instance().info("Heartbeat: connection restored during retry");
            m_wasConnected = true;
            emit connectionRestored();
        } else {
            if (!m_wasConnected) {
                m_wasConnected = true;
                Logger::instance().info("Heartbeat: connection restored");
                emit connectionRestored();
            } else {
                Logger::instance().debug("Heartbeat: connection alive");
                emit connectionAlive();
            }
        }
    } else {
        m_consecutiveFailures++;

        Logger::instance().warning(
            QString("Heartbeat: check failed (%1/%2) - %3")
                .arg(m_consecutiveFailures)
                .arg(FAILURE_THRESHOLD)
                .arg(reply->errorString()));

        if (!m_retrying && m_consecutiveFailures >= FAILURE_THRESHOLD) {
            Logger::instance().warning("Heartbeat: connection lost, starting retry sequence");
            m_wasConnected = false;
            m_checkTimer->stop();
            emit connectionLost();
            startRetrySequence();
        }
    }

    reply->deleteLater();
}

void Heartbeat::startRetrySequence()
{
    m_retrying = true;
    m_retryCount = 0;
    m_retryDelay = RETRY_DELAYS.first();

    Logger::instance().info(QString("Heartbeat: first retry in %1ms").arg(m_retryDelay));
    m_retryTimer->start(m_retryDelay);
}

void Heartbeat::onRetryTimeout()
{
    if (!m_retrying)
        return;

    Logger::instance().info(
        QString("Heartbeat: retry attempt %1/%2").arg(m_retryCount + 1).arg(MAX_RETRIES));

    m_retryCount++;

    emit reconnectRequested();

    if (m_retryCount >= MAX_RETRIES) {
        Logger::instance().error("Heartbeat: all retries exhausted");
        m_retrying = false;
        m_checkTimer->start();
        return;
    }

    if (m_retryCount < RETRY_DELAYS.size()) {
        m_retryDelay = RETRY_DELAYS.at(m_retryCount);
    } else {
        m_retryDelay = RETRY_DELAYS.last();
    }

    m_retryTimer->start(m_retryDelay);
}

void Heartbeat::sendHeadRequest(const QString& url)
{
    cancelActiveRequest();

    QNetworkRequest request{QUrl(url)};
    request.setTransferTimeout(REQUEST_TIMEOUT);
    request.setAttribute(QNetworkRequest::RedirectPolicyAttribute,
                         QNetworkRequest::NoLessSafeRedirectPolicy);

    m_activeReply = m_networkManager->head(request);
}

void Heartbeat::resetRetryState()
{
    m_retrying = false;
    m_retryCount = 0;
    m_retryTimer->stop();
}

void Heartbeat::cancelActiveRequest()
{
    if (m_activeReply) {
        m_activeReply->abort();
        m_activeReply->deleteLater();
        m_activeReply = nullptr;
    }
}