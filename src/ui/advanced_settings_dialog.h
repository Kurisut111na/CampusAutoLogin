#pragma once

#include <QDialog>
#include <QSpinBox>
#include <QCheckBox>
#include <QLineEdit>
#include <QLabel>
#include <QTextEdit>

struct AppConfig;

class AdvancedSettingsDialog : public QDialog
{
    Q_OBJECT

public:
    explicit AdvancedSettingsDialog(QWidget* parent = nullptr);

    void loadFromConfig(const AppConfig& config);
    void saveToConfig(AppConfig& config);

private:
    void setupUi();

    QSpinBox*   m_heartbeatIntervalSpin;
    QCheckBox*  m_heartbeatEnableCheck;
    QCheckBox*  m_autoStartCheck;
    QCheckBox*  m_startMinimizedCheck;
    QLineEdit*  m_customGatewayEdit;
    QTextEdit*  m_pingUrlsEdit;
    QLabel*     m_resourceNote;
    QLabel*     m_logDirLabel;
};