#include "login_manager.h"
#include "logger.h"

#include <QNetworkRequest>

LoginManager::LoginManager(QObject* parent)
    : QObject(parent)
    , m_networkManager(new QNetworkAccessManager(this))
{
    connect(m_networkManager, &QNetworkAccessManager::finished,
            this, &LoginManager::onLoginReplyFinished);
}

void LoginManager::login(const QString& username,
                          const QString& password,
                          const QString& operator_,
                          const QString& localIp,
                          const QString& gateway)
{
    if (m_activeReply) {
        Logger::instance().warning("Login already in progress, aborting previous request");
        abortLogin();
    }

    QUrl url = buildLoginUrl(username, password, operator_, localIp, gateway);
    Logger::instance().info(QString("Sending login request to: %1").arg(url.toString()));

    QNetworkRequest request(url);
    request.setHeader(QNetworkRequest::ContentTypeHeader, "application/x-www-form-urlencoded");
    request.setAttribute(QNetworkRequest::RedirectPolicyAttribute,
                         QNetworkRequest::NoLessSafeRedirectPolicy);
    request.setTransferTimeout(10000);

    m_activeReply = m_networkManager->get(request);
}

void LoginManager::abortLogin()
{
    if (m_activeReply) {
        m_activeReply->abort();
        m_activeReply->deleteLater();
        m_activeReply = nullptr;
    }
}

bool LoginManager::isLoggedIn() const
{
    return m_loggedIn;
}

void LoginManager::onLoginReplyFinished(QNetworkReply* reply)
{
    if (reply != m_activeReply)
        return;

    m_activeReply = nullptr;

    if (reply->error() != QNetworkReply::NoError) {
        QString errorMsg = QString("Network error: %1").arg(reply->errorString());
        Logger::instance().error(errorMsg);
        m_loggedIn = false;
        emit loginFailed(errorMsg);
        reply->deleteLater();
        return;
    }

    QByteArray responseData = reply->readAll();
    QString responseText = QString::fromUtf8(responseData);
    Logger::instance().info(QString("Server response: %1").arg(responseText.left(500)));

    int statusCode = reply->attribute(QNetworkRequest::HttpStatusCodeAttribute).toInt();
    Logger::instance().info(QString("HTTP status code: %1").arg(statusCode));

    if (statusCode == 200) {
        m_loggedIn = true;
        Logger::instance().info("Login successful");
        emit loginSuccess();
    } else {
        m_loggedIn = false;
        QString err = QString("Login failed with HTTP %1: %2")
                          .arg(statusCode)
                          .arg(responseText.left(200));
        Logger::instance().error(err);
        emit loginFailed(err);
    }

    reply->deleteLater();
}

QUrl LoginManager::buildLoginUrl(const QString& username,
                                  const QString& password,
                                  const QString& operator_,
                                  const QString& localIp,
                                  const QString& gateway) const
{
    QString ddddd = QUrl::toPercentEncoding(username + "@" + operator_);
    QString encodedPassword = QUrl::toPercentEncoding(password);
    QString encodedIp = QUrl::toPercentEncoding(localIp);

    QString fullUrl = QString(
        "http://%4/drcom/login?"
        "callback=dr1004&"
        "DDDDD=%1&"
        "upass=%2&"
        "0MKKey=123456&"
        "R1=0&R2=&R3=0&R6=1&"
        "para=00&"
        "v4ip=%3&"
        "v6ip=&"
        "terminal_type=2&"
        "lang=zh-cn&"
        "jsVersion=4.2&"
        "v=608"
    ).arg(ddddd, encodedPassword, encodedIp, gateway);

    return QUrl(fullUrl);
}