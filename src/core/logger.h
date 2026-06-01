#pragma once

#include <QObject>
#include <QString>
#include <QFile>
#include <QTextStream>
#include <QMutex>
#include <QDateTime>
#include <QStringList>
#include <QList>

enum class LogLevel {
    Debug,
    Info,
    Warning,
    Error
};

struct LogEntry {
    QDateTime timestamp;
    LogLevel level;
    QString message;
};

class Logger : public QObject
{
    Q_OBJECT

public:
    static Logger& instance();

    void setLogLevel(LogLevel level);
    LogLevel logLevel() const;

    void debug(const QString& message);
    void info(const QString& message);
    void warning(const QString& message);
    void error(const QString& message);

    QString logDir() const;
    QList<LogEntry> recentEntries(int count = 50) const;

    void clearRecentEntries();

signals:
    void newLogEntry(const LogEntry& entry);

private:
    Logger();
    ~Logger() override;
    Logger(const Logger&) = delete;
    Logger& operator=(const Logger&) = delete;

    void writeLog(LogLevel level, const QString& message);
    void openLogFile();
    void cleanupOldLogs();
    QString levelToString(LogLevel level) const;

    QString m_logDir;
    QFile m_logFile;
    QTextStream m_stream;
    mutable QMutex m_mutex;
    LogLevel m_minLevel = LogLevel::Info;

    static constexpr int MAX_RECENT_ENTRIES = 100;
    QList<LogEntry> m_recentEntries;
};