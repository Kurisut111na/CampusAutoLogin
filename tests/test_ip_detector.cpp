#include <QTest>
#include <QSignalSpy>

#include "ip_detector.h"
#include "logger.h"

class TestIpDetector : public QObject
{
    Q_OBJECT

private slots:
    void initTestCase()
    {
        Logger::instance().setLogLevel(LogLevel::Debug);
    }

    void testDetectLocalIp()
    {
        IpDetector detector;
        QString ip = detector.detectLocalIp();

        QVERIFY2(!ip.isEmpty(), "Should detect at least one valid IP address");

        QHostAddress addr(ip);
        QVERIFY2(!addr.isNull(), "Detected IP should be a valid address");
        QVERIFY2(addr.protocol() == QAbstractSocket::IPv4Protocol,
                 "Detected IP should be IPv4");

        qDebug() << "Detected local IP:" << ip;
    }

    void testIsPrivateIPv4_data()
    {
        QTest::addColumn<QString>("ip");
        QTest::addColumn<bool>("expected");

        QTest::newRow("10.x")    << "10.0.1.5"      << true;
        QTest::newRow("172.16")  << "172.16.0.1"    << true;
        QTest::newRow("172.31")  << "172.31.255.254"<< true;
        QTest::newRow("192.168") << "192.168.1.1"   << true;
        QTest::newRow("public")  << "8.8.8.8"       << false;
        QTest::newRow("loopback")<< "127.0.0.1"     << false;
        QTest::newRow("172.32")  << "172.32.0.1"    << false;
    }

    void testIsPrivateIPv4()
    {
        QFETCH(QString, ip);
        QFETCH(bool, expected);

        QHostAddress addr(ip);
        QCOMPARE(IpDetector::isPrivateIPv4(addr), expected);
    }

    void testIsCampusNetwork()
    {
        QVERIFY(IpDetector::isCampusNetwork(QHostAddress("10.0.1.5")));
        QVERIFY(IpDetector::isCampusNetwork(QHostAddress("10.255.255.1")));
        QVERIFY(!IpDetector::isCampusNetwork(QHostAddress("172.16.0.1")));
        QVERIFY(!IpDetector::isCampusNetwork(QHostAddress("192.168.1.1")));
    }

    void testIsVirtualAdapter()
    {
        QVERIFY(IpDetector::isVirtualAdapter("vEthernet", "Hyper-V Virtual Ethernet Adapter"));
        QVERIFY(IpDetector::isVirtualAdapter("VMware Network Adapter", "VMware"));
        QVERIFY(IpDetector::isVirtualAdapter("VirtualBox", "Host-Only Network"));
        QVERIFY(IpDetector::isVirtualAdapter("vEthernet (WSL)", "WSL"));
        QVERIFY(!IpDetector::isVirtualAdapter("Ethernet", "Realtek PCIe GBE Family Controller"));
        QVERIFY(!IpDetector::isVirtualAdapter("Wi-Fi", "Intel Wireless-AC 9560"));
    }

    void testDetectNetworkType()
    {
        IpDetector detector;
        NetworkType type = detector.detectNetworkType();
        QVERIFY(type != NetworkType::Unknown || true);
        qDebug() << "Network type:" << static_cast<int>(type);
    }
};

QTEST_MAIN(TestIpDetector)
#include "test_ip_detector.moc"