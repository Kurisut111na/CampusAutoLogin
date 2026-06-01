#pragma once

#include <QSystemTrayIcon>
#include <QMenu>
#include <QAction>
#include <QIcon>

class TrayIcon : public QSystemTrayIcon
{
    Q_OBJECT

public:
    explicit TrayIcon(QObject* parent = nullptr);

    void setLoggedIn(bool loggedIn);
    void setConnectionStatus(bool connected);

signals:
    void showRequested();
    void reconnectRequested();
    void openLogDirRequested();
    void quitRequested();

private:
    void updateIcon();
    void setupMenu();

    QMenu* m_menu;
    QAction* m_showAction;
    QAction* m_reconnectAction;
    QAction* m_openLogDirAction;
    QAction* m_quitAction;

    bool m_loggedIn = false;
    bool m_connected = false;

    QIcon m_iconConnected;
    QIcon m_iconDisconnected;
    QIcon m_iconLoggedIn;
};