#include "logger.h"

#include <QCoreApplication>
#include <QStandardPaths>
#include <QDir>
#include <QFileInfo>

Logger& Logger::instance()
{
    static Logger* s_instance = new Logger();
    return *s_instance;
}

Logger::Logger()
    : QObject(nullptr)
{
    m_logDir = QStandardPaths::writableLocation(QStandardPaths::AppLocalDataLocation)
               + "/logs";
    QDir().mkpath(m_logDir);

    cleanupOldLogs();
    openLogFile();
}

Logger::~Logger()
{
    m_stream.flush();
    m_logFile.close();
}

void Logger::setLogLevel(LogLevel level)
{
    m_minLevel = level;
}

LogLevel Logger::logLevel() const
{
    return m_minLevel;
}

QString Logger::logDir() const
{
    return m_logDir;
}

QList<LogEntry> Logger::recentEntries(int count) const
{
    QMutexLocker locker(&m_mutex);
    if (count >= m_recentEntries.size())
        return m_recentEntries;
    return m_recentEntries.mid(m_recentEntries.size() - count);
}

void Logger::clearRecentEntries()
{
    QMutexLocker locker(&m_mutex);
    m_recentEntries.clear();
}

void Logger::debug(const QString& message)
{
    writeLog(LogLevel::Debug, message);
}

void Logger::info(const QString& message)
{
    writeLog(LogLevel::Info, message);
}

void Logger::warning(const QString& message)
{
    writeLog(LogLevel::Warning, message);
}

void Logger::error(const QString& message)
{
    writeLog(LogLevel::Error, message);
}

void Logger::writeLog(LogLevel level, const QString& message)
{
    if (level < m_minLevel)
        return;

    QDateTime now = QDateTime::currentDateTime();

    {
        QMutexLocker locker(&m_mutex);

        QString timestamp = now.toString("yyyy-MM-dd hh:mm:ss");
        QString levelStr = QString("%1").arg(levelToString(level), -7);
        QString line = QString("[%1] %2 %3\n").arg(timestamp, levelStr, message);
        m_stream << line;
        m_stream.flush();

        LogEntry entry;
        entry.timestamp = now;
        entry.level = level;
        entry.message = message;
        m_recentEntries.append(entry);
        while (m_recentEntries.size() > MAX_RECENT_ENTRIES) {
            m_recentEntries.removeFirst();
        }
    }

    emit newLogEntry({now, level, message});

    QString todayStr = QDate::currentDate().toString("yyyy-MM-dd");
    QString currentLogFile = m_logFile.fileName();
    if (!currentLogFile.contains(todayStr)) {
        openLogFile();
    }
}

void Logger::openLogFile()
{
    if (m_logFile.isOpen()) {
        m_stream.flush();
        m_logFile.close();
    }

    QString today = QDate::currentDate().toString("yyyy-MM-dd");
    QString fileName = m_logDir + "/campus-autologin-" + today + ".log";

    m_logFile.setFileName(fileName);
    m_logFile.open(QIODevice::WriteOnly | QIODevice::Append | QIODevice::Text);
    m_stream.setDevice(&m_logFile);
}

void Logger::cleanupOldLogs()
{
    QDir dir(m_logDir);
    QStringList filters;
    filters << "campus-autologin-*.log";
    QFileInfoList files = dir.entryInfoList(filters, QDir::Files, QDir::Name);

    QDate cutoff = QDate::currentDate().addDays(-7);

    for (const QFileInfo& fileInfo : files) {
        QString name = fileInfo.baseName();
        QString dateStr = name.mid(QString("campus-autologin-").length());
        QDate fileDate = QDate::fromString(dateStr, "yyyy-MM-dd");
        if (fileDate.isValid() && fileDate < cutoff) {
            QFile::remove(fileInfo.absoluteFilePath());
        }
    }
}

QString Logger::levelToString(LogLevel level) const
{
    switch (level) {
    case LogLevel::Debug:   return "DEBUG";
    case LogLevel::Info:    return "INFO";
    case LogLevel::Warning: return "WARN";
    case LogLevel::Error:   return "ERROR";
    }
    return "UNKNOWN";
}