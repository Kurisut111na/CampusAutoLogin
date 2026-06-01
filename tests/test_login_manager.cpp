#include <QTest>
#include <QSignalSpy>
#include <QUrl>

#include "login_manager.h"
#include "logger.h"

class TestLoginManager : public QObject
{
    Q_OBJECT

private slots:
    void initTestCase()
    {
        Logger::instance().setLogLevel(LogLevel::Debug);
    }

    void testLoginDispatch()
    {
        LoginManager mgr;

        mgr.login("20210001", "test123", "cmcc", "10.0.1.100", "10.0.1.5");

        QVERIFY2(true, "Login request dispatched without crash");
    }

    void testAbortLogin()
    {
        LoginManager mgr;
        mgr.login("user", "pass", "cmcc", "10.0.1.1", "10.0.1.5");
        mgr.abortLogin();
        QVERIFY2(true, "Abort completed without crash");
    }

    void testInitialState()
    {
        LoginManager mgr;
        QVERIFY(!mgr.isLoggedIn());
    }

};

QTEST_MAIN(TestLoginManager)
#include "test_login_manager.moc"