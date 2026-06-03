package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogLevel represents the severity of a log entry.
type LogLevel int

const (
	LogDebug   LogLevel = 0
	LogInfo    LogLevel = 1
	LogWarning LogLevel = 2
	LogError   LogLevel = 3
)

func (l LogLevel) String() string {
	switch l {
	case LogDebug:
		return "DEBUG"
	case LogInfo:
		return "INFO"
	case LogWarning:
		return "WARN"
	case LogError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// LogEntry is a single log record.
type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Message   string
}

// Logger is a thread-safe file logger.
type Logger struct {
	mu        sync.Mutex
	logDir    string
	file      *os.File
	minLevel  LogLevel
	recent    []LogEntry
	maxRecent int
}

var logger *Logger

// InitLogger creates the log directory and opens the current log file.
func InitLogger(minLevel LogLevel) error {
	localAppData, err := os.UserCacheDir()
	if err != nil {
		// Fallback: use LocalAppData
		localAppData = os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
		}
	} else {
		// UserCacheDir returns AppData/Local on Windows, but check
		localAppData = os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
		}
	}

	logDir := filepath.Join(localAppData, "CampusAutoLogin", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	logFile := filepath.Join(logDir, time.Now().Format("2006-01-02")+".log")
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	logger = &Logger{
		logDir:    logDir,
		file:      f,
		minLevel:  minLevel,
		recent:    make([]LogEntry, 0, 200),
		maxRecent: 200,
	}

	// Clean old logs
	go logger.cleanOldLogs()

	return nil
}

// GetLogger returns the global logger instance.
func GetLogger() *Logger {
	if logger == nil {
		// Fallback: create a minimal stderr logger
		_ = InitLogger(LogDebug)
	}
	return logger
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.minLevel {
		return
	}

	msg := fmt.Sprintf(format, args...)
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   msg,
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Write to file
	line := fmt.Sprintf("[%s] [%s] %s\n",
		entry.Timestamp.Format("2006-01-02 15:04:05.000"),
		entry.Level.String(),
		entry.Message,
	)
	if l.file != nil {
		l.file.WriteString(line)
		l.file.Sync()
	}

	// Store recent
	l.recent = append(l.recent, entry)
	if len(l.recent) > l.maxRecent {
		l.recent = l.recent[len(l.recent)-l.maxRecent:]
	}
}

func (l *Logger) Debug(format string, args ...interface{}) { l.log(LogDebug, format, args...) }
func (l *Logger) Info(format string, args ...interface{})  { l.log(LogInfo, format, args...) }
func (l *Logger) Warn(format string, args ...interface{})  { l.log(LogWarning, format, args...) }
func (l *Logger) Error(format string, args ...interface{}) { l.log(LogError, format, args...) }

// RecentEntries returns the most recent log entries.
func (l *Logger) RecentEntries(count int) []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	if count <= 0 || count > len(l.recent) {
		count = len(l.recent)
	}
	start := len(l.recent) - count
	result := make([]LogEntry, count)
	copy(result, l.recent[start:])
	return result
}

// AllRecentEntries returns all buffered log entries.
func (l *Logger) AllRecentEntries() []LogEntry {
	return l.RecentEntries(l.maxRecent)
}

// LogDir returns the log directory path.
func (l *Logger) LogDir() string { return l.logDir }

// CleanRecent clears the in-memory log buffer.
func (l *Logger) CleanRecent() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.recent = l.recent[:0]
}

func (l *Logger) cleanOldLogs() {
	l.mu.Lock()
	dir := l.logDir
	l.mu.Unlock()

	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Parse date from filename like "2006-01-02.log"
		t, err := time.Parse("2006-01-02.log", name)
		if err != nil {
			continue
		}
		if t.Before(cutoff) {
			os.Remove(filepath.Join(dir, name))
		}
	}
}

// Close closes the log file.
func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		l.file.Close()
		l.file = nil
	}
}
