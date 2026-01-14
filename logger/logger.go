package logger

import (
	"fmt"
	"io"
	"os"
	"time"
)

// LogLevel defines the severity of the log
type LogLevel int

const (
	LogLevelSilent LogLevel = iota
	LogLevelError
	LogLevelWarn
	LogLevelInfo
)

// Logger is the interface for logging SQL and internal messages
type Logger interface {
	SetLevel(level LogLevel)
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
	SQL(sql string, duration time.Duration, args ...any)
}

// StdLogger is a basic implementation of Logger using standard output
type stdLogger struct {
	level  LogLevel
	writer io.Writer
}

// NewStdLogger creates a new standard logger
func NewStdLogger() Logger {
	return &stdLogger{
		level:  LogLevelInfo,
		writer: os.Stdout,
	}
}

func (l *stdLogger) SetLevel(level LogLevel) {
	l.level = level
}

func (l *stdLogger) Info(format string, args ...any) {
	if l.level >= LogLevelInfo {
		l.log("INFO", format, args...)
	}
}

func (l *stdLogger) Warn(format string, args ...any) {
	if l.level >= LogLevelWarn {
		l.log("WARN", format, args...)
	}
}

func (l *stdLogger) Error(format string, args ...any) {
	if l.level >= LogLevelError {
		l.log("ERROR", format, args...)
	}
}

func (l *stdLogger) SQL(sql string, duration time.Duration, args ...any) {
	if l.level >= LogLevelInfo {
		l.log("SQL", "[%v] %s | args: %v", duration, sql, args)
	}
}

func (l *stdLogger) log(level, format string, args ...any) {
	fmt.Fprintf(l.writer, "[JORM] %s %s: %s\n", time.Now().Format("2006-01-02 15:04:05"), level, fmt.Sprintf(format, args...))
}
