package internal

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// LogLevel represents the logging level
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// String returns the string representation of the log level
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "INFO"
	}
}

// LoggerInterface defines the logging contract
type LoggerInterface interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	SetLevel(level LogLevel)
}

// DefaultLogger is a basic implementation of the Logger interface
type DefaultLogger struct {
	level  LogLevel
	logger *log.Logger
}

// NewDefaultLogger creates a new default logger with the specified level
func NewDefaultLogger(levelStr string) LoggerInterface {
	level := parseLogLevel(levelStr)
	return &DefaultLogger{
		level:  level,
		logger: log.New(os.Stdout, "[HEALER] ", log.LstdFlags),
	}
}

// parseLogLevel converts a string to LogLevel
func parseLogLevel(levelStr string) LogLevel {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		return LogLevelDebug
	case "INFO":
		return LogLevelInfo
	case "WARN", "WARNING":
		return LogLevelWarn
	case "ERROR":
		return LogLevelError
	default:
		return LogLevelInfo // default to info
	}
}

// Debug logs a debug message
func (l *DefaultLogger) Debug(msg string, args ...any) {
	if l.level <= LogLevelDebug {
		l.log(LogLevelDebug, msg, args...)
	}
}

// Info logs an info message
func (l *DefaultLogger) Info(msg string, args ...any) {
	if l.level <= LogLevelInfo {
		l.log(LogLevelInfo, msg, args...)
	}
}

// Warn logs a warning message
func (l *DefaultLogger) Warn(msg string, args ...any) {
	if l.level <= LogLevelWarn {
		l.log(LogLevelWarn, msg, args...)
	}
}

// Error logs an error message
func (l *DefaultLogger) Error(msg string, args ...any) {
	if l.level <= LogLevelError {
		l.log(LogLevelError, msg, args...)
	}
}

// SetLevel sets the logging level
func (l *DefaultLogger) SetLevel(level LogLevel) {
	l.level = level
}

// log is the internal logging method
func (l *DefaultLogger) log(level LogLevel, msg string, args ...any) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelStr := level.String()

	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	l.logger.Printf("%s [%s] %s", timestamp, levelStr, msg)
}
