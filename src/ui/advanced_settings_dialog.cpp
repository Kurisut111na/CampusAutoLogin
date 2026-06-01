#include "advanced_settings_dialog.h"
#include "config_manager.h"
#include "logger.h"

#include <QVBoxLayout>
#include <QFormLayout>
#include <QGroupBox>
#include <QDialogButtonBox>
#include <QLabel>
#include <QPushButton>

AdvancedSettingsDialog::AdvancedSettingsDialog(QWidget* parent)
    : QDialog(parent)
{
    setupUi();
}

void AdvancedSettingsDialog::setupUi()
{
    setWindowTitle("高级设置");
    setMinimumSize(460, 420);

    QVBoxLayout* mainLayout = new QVBoxLayout(this);

    QGroupBox* networkGroup = new QGroupBox("网络设置");
    QFormLayout* networkLayout = new QFormLayout(networkGroup);

    m_customGatewayEdit = new QLineEdit;
    m_customGatewayEdit->setPlaceholderText("留空则自动探测");
    m_customGatewayEdit->setToolTip("手动指定 Dr.COM 网关 IP（例如 10.0.1.5）");
    networkLayout->addRow("自定义网关：", m_customGatewayEdit);

    mainLayout->addWidget(networkGroup);

    QGroupBox* heartbeatGroup = new QGroupBox("心跳检测");
    QVBoxLayout* heartbeatLayout = new QVBoxLayout(heartbeatGroup);

    m_heartbeatEnableCheck = new QCheckBox("启用心跳检测");
    heartbeatLayout->addWidget(m_heartbeatEnableCheck);

    QHBoxLayout* intervalLayout = new QHBoxLayout;
    intervalLayout->addWidget(new QLabel("检测间隔："));
    m_heartbeatIntervalSpin = new QSpinBox;
    m_heartbeatIntervalSpin->setRange(15, 300);
    m_heartbeatIntervalSpin->setValue(45);
    m_heartbeatIntervalSpin->setMinimumWidth(120);
    m_heartbeatIntervalSpin->setSuffix(" 秒");
    m_heartbeatIntervalSpin->setToolTip("每隔多少秒检测一次网络连通性（15 ~ 300 秒）");
    intervalLayout->addWidget(m_heartbeatIntervalSpin);
    QLabel* intervalHint = new QLabel("范围 15 ~ 300 秒");
    intervalHint->setStyleSheet("color: #888; font-size: 11px;");
    intervalLayout->addWidget(intervalHint);
    intervalLayout->addStretch();
    heartbeatLayout->addLayout(intervalLayout);

    heartbeatLayout->addWidget(new QLabel("检测网址（每行一个）："));
    m_pingUrlsEdit = new QTextEdit;
    m_pingUrlsEdit->setPlaceholderText("https://www.baidu.com\nhttps://www.bing.com");
    m_pingUrlsEdit->setMaximumHeight(80);
    m_pingUrlsEdit->setToolTip("心跳检测会轮流访问这些网址来判断网络是否连通");
    heartbeatLayout->addWidget(m_pingUrlsEdit);

    m_resourceNote = new QLabel("<span style='color:#888;font-size:11px;'>"
        "说明：心跳检测通过 HEAD 请求（仅获取响应头，不下载内容）<br>"
        "轮流检测上述网址。每次请求数据量约 1KB，对网络和系统资源<br>"
        "占用极低（CPU &lt;0.1%，内存约 1MB），可放心开启。</span>");
    m_resourceNote->setWordWrap(true);
    heartbeatLayout->addWidget(m_resourceNote);

    mainLayout->addWidget(heartbeatGroup);

    QGroupBox* systemGroup = new QGroupBox("系统");
    QVBoxLayout* systemLayout = new QVBoxLayout(systemGroup);

    m_autoStartCheck = new QCheckBox("开机自启动");
    systemLayout->addWidget(m_autoStartCheck);

    m_startMinimizedCheck = new QCheckBox("启动后最小化到托盘（不显示窗口）");
    m_startMinimizedCheck->setToolTip("勾选后，启动时若自动登录条件满足，窗口将不弹出，仅显示托盘图标");
    systemLayout->addWidget(m_startMinimizedCheck);

    QHBoxLayout* logDirLayout = new QHBoxLayout;
    m_logDirLabel = new QLabel(Logger::instance().logDir());
    m_logDirLabel->setStyleSheet("color: #666; font-size: 11px;");
    m_logDirLabel->setWordWrap(true);
    logDirLayout->addWidget(new QLabel("日志目录："));
    logDirLayout->addWidget(m_logDirLabel, 1);
    systemLayout->addLayout(logDirLayout);

    mainLayout->addWidget(systemGroup);

    mainLayout->addStretch();

    QDialogButtonBox* buttonBox = new QDialogButtonBox(
        QDialogButtonBox::Ok | QDialogButtonBox::Cancel);
    buttonBox->button(QDialogButtonBox::Ok)->setText("确定");
    buttonBox->button(QDialogButtonBox::Cancel)->setText("取消");
    connect(buttonBox, &QDialogButtonBox::accepted, this, &QDialog::accept);
    connect(buttonBox, &QDialogButtonBox::rejected, this, &QDialog::reject);
    mainLayout->addWidget(buttonBox);
}

void AdvancedSettingsDialog::loadFromConfig(const AppConfig& config)
{
    m_heartbeatEnableCheck->setChecked(config.heartbeatEnabled);
    m_heartbeatIntervalSpin->setValue(config.heartbeatInterval);
    m_autoStartCheck->setChecked(config.autoStart);
    m_startMinimizedCheck->setChecked(config.startMinimized);
    m_customGatewayEdit->setText(config.customGateway);
    m_pingUrlsEdit->setPlainText(config.pingUrls.join("\n"));
    m_logDirLabel->setText(Logger::instance().logDir());
}

void AdvancedSettingsDialog::saveToConfig(AppConfig& config)
{
    config.heartbeatEnabled = m_heartbeatEnableCheck->isChecked();
    config.heartbeatInterval = m_heartbeatIntervalSpin->value();
    config.autoStart = m_autoStartCheck->isChecked();
    config.startMinimized = m_startMinimizedCheck->isChecked();
    config.customGateway = m_customGatewayEdit->text().trimmed();
    config.pingUrls = m_pingUrlsEdit->toPlainText().split("\n", Qt::SkipEmptyParts);
    for (auto& url : config.pingUrls)
        url = url.trimmed();
    config.pingUrls.removeAll({});
}