#pragma once

#include <QObject>
#include <QNetworkAccessManager>
#include <QNetworkReply>
#include <QUrl>

class LoginManager : public QObject
{
    Q_OBJECT

public:
    explicit LoginManager(QObject* parent = nullptr);

    void login(const QString& username,
               const QString& password,
               const QString& operator_,
               const QString& localIp,
               const QString& gateway);

    void abortLogin();

    bool isLoggedIn() const;

signals:
    void loginSuccess();
    void loginFailed(const QString& errorMessage);

private slots:
    void onLoginReplyFinished(QNetworkReply* reply);

private:
    QUrl buildLoginUrl(const QString& username,
                       const QString& password,
                       const QString& operator_,
                       const QString& localIp,
                       const QString& gateway) const;

    QNetworkAccessManager* m_networkManager;
    QNetworkReply* m_activeReply = nullptr;
    bool m_loggedIn = false;
};