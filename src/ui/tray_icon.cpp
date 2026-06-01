#include "tray_icon.h"
#include "logger.h"

#include <QApplication>
#include <QStyle>
#include <QPainter>
#include <QPixmap>
#include <QDesktopServices>
#include <QUrl>

static QIcon createColoredIcon(const QColor& color)
{
    QPixmap pixmap(32, 32);
    pixmap.fill(Qt::transparent);

    QPainter painter(&pixmap);
    painter.setRenderHint(QPainter::Antialiasing);
    painter.setBrush(color);
    painter.setPen(Qt::NoPen);
    painter.drawEllipse(4, 4, 24, 24);

    painter.setPen(Qt::white);
    QFont font;
    font.setPixelSize(14);
    font.setBold(true);
    painter.setFont(font);
    painter.drawText(pixmap.rect(), Qt::AlignCenter, "N");

    painter.end();
    return QIcon(pixmap);
}

TrayIcon::TrayIcon(QObject* parent)
    : QSystemTrayIcon(parent)
{
    m_iconConnected = createColoredIcon(QColor("#4CAF50"));
    m_iconDisconnected = createColoredIcon(QColor("#f44336"));
    m_iconLoggedIn = createColoredIcon(QColor("#2196F3"));

    updateIcon();
    setupMenu();

    show();
    Logger::instance().info("Tray icon created");
}

void TrayIcon::setLoggedIn(bool loggedIn)
{
    m_loggedIn = loggedIn;
    updateIcon();
    m_reconnectAction->setEnabled(loggedIn);
}

void TrayIcon::setConnectionStatus(bool connected)
{
    m_connected = connected;
    if (!m_loggedIn)
        updateIcon();
}

void TrayIcon::updateIcon()
{
    if (m_loggedIn) {
        setIcon(m_iconLoggedIn);
        setToolTip("校园网自动登录 - 已登录");
    } else if (m_connected) {
        setIcon(m_iconConnected);
        setToolTip("校园网自动登录 - 已连接");
    } else {
        setIcon(m_iconDisconnected);
        setToolTip("校园网自动登录 - 已断开");
    }
}

void TrayIcon::setupMenu()
{
    m_menu = new QMenu;

    m_showAction = m_menu->addAction("显示窗口");
    connect(m_showAction, &QAction::triggered, this, &TrayIcon::showRequested);

    m_menu->addSeparator();

    m_reconnectAction = m_menu->addAction("立即重连");
    m_reconnectAction->setEnabled(false);
    connect(m_reconnectAction, &QAction::triggered, this, &TrayIcon::reconnectRequested);

    m_openLogDirAction = m_menu->addAction("打开日志目录");
    connect(m_openLogDirAction, &QAction::triggered, this, &TrayIcon::openLogDirRequested);

    m_menu->addSeparator();

    m_quitAction = m_menu->addAction("退出");
    connect(m_quitAction, &QAction::triggered, this, &TrayIcon::quitRequested);

    setContextMenu(m_menu);
}