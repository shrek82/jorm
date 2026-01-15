package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const (
	ansiReset   = "\033[0m"
	ansiRed     = "\033[31m"
	ansiGreen   = "\033[32m"
	ansiYellow  = "\033[33m"
	ansiBlue    = "\033[34m"
	ansiMagenta = "\033[35m"
	ansiCyan    = "\033[36m"
)

// LogLevel defines the severity of the log
type LogLevel int

const (
	LogLevelSilent LogLevel = iota
	LogLevelError
	LogLevelWarn
	LogLevelInfo
)

// LogFormat defines the output format of the log
type LogFormat string

const (
	LogFormatText LogFormat = "text"
	LogFormatJSON LogFormat = "json"
)

// Logger is the interface for logging SQL and internal messages
type Logger interface {
	SetLevel(level LogLevel)
	SetFormat(format LogFormat)
	SetOutput(w io.Writer)
	SetLevelOutput(level LogLevel, w io.Writer)
	WithFields(fields map[string]any) Logger
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
	SQL(sql string, duration time.Duration, args ...any)
}

// baseLogger contains common logging functionality
type baseLogger struct {
	level        LogLevel
	format       LogFormat
	writer       io.Writer
	levelWriters map[LogLevel]io.Writer
	fields       map[string]any
}

func (l *baseLogger) SetLevel(level LogLevel) {
	l.level = level
}

func (l *baseLogger) SetFormat(format LogFormat) {
	l.format = format
}

func (l *baseLogger) SetOutput(w io.Writer) {
	l.writer = w
}

func (l *baseLogger) SetLevelOutput(level LogLevel, w io.Writer) {
	if l.levelWriters == nil {
		l.levelWriters = make(map[LogLevel]io.Writer)
	}
	l.levelWriters[level] = w
}

func (l *baseLogger) clone() *baseLogger {
	newFields := make(map[string]any, len(l.fields))
	for k, v := range l.fields {
		newFields[k] = v
	}
	newLevelWriters := make(map[LogLevel]io.Writer, len(l.levelWriters))
	for k, v := range l.levelWriters {
		newLevelWriters[k] = v
	}
	return &baseLogger{
		level:        l.level,
		format:       l.format,
		writer:       l.writer,
		levelWriters: newLevelWriters,
		fields:       newFields,
	}
}

// stdLogger is the default implementation of Logger
type stdLogger struct {
	baseLogger
}

// NewStdLogger creates a new standard logger
func NewStdLogger() Logger {
	return &stdLogger{
		baseLogger: baseLogger{
			level:        LogLevelInfo,
			format:       LogFormatText,
			writer:       os.Stdout,
			levelWriters: make(map[LogLevel]io.Writer),
			fields:       make(map[string]any),
		},
	}
}

func (l *stdLogger) WithFields(fields map[string]any) Logger {
	newLogger := &stdLogger{
		baseLogger: *l.clone(),
	}
	for k, v := range fields {
		newLogger.fields[k] = v
	}
	return newLogger
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
		if l.format == LogFormatJSON {
			l.log("SQL", "", "sql", sql, "duration", duration.String(), "args", args)
		} else {
			l.log("SQL", "[%v] %s | args: %v", duration, sql, args)
		}
	}
}

func (l *stdLogger) log(level string, format string, args ...any) {
	now := time.Now()
	msgLevel := l.parseLevel(level)

	// Determine all writers for this message
	var writers []io.Writer
	if l.writer != nil {
		writers = append(writers, l.writer)
	}
	if w, ok := l.levelWriters[msgLevel]; ok && w != nil {
		// If it's the same as default writer, don't duplicate
		if w != l.writer {
			writers = append(writers, w)
		}
	}

	if len(writers) == 0 {
		return
	}

	if l.format == LogFormatJSON {
		data := make(map[string]any)
		for k, v := range l.fields {
			data[k] = v
		}
		data["time"] = now.Format(time.RFC3339)
		data["level"] = level
		if format != "" {
			if len(args) > 0 {
				data["msg"] = fmt.Sprintf(format, args...)
			} else {
				data["msg"] = format
			}
		} else if len(args) >= 2 {
			// Handle structured fields passed as args for SQL log or similar
			for i := 0; i < len(args); i += 2 {
				if i+1 < len(args) {
					if key, ok := args[i].(string); ok {
						data[key] = args[i+1]
					}
				}
			}
		}

		jsonData, _ := json.Marshal(data)
		for _, w := range writers {
			w.Write(append(jsonData, '\n'))
		}
	} else {
		msg := ""
		if format != "" {
			msg = fmt.Sprintf(format, args...)
		}

		if level == "SQL" && len(args) >= 2 {
			if sqlStr, ok := args[1].(string); ok {
				color := getSQLColor(sqlStr)
				msg = fmt.Sprintf("%s%s%s", color, msg, ansiReset)
			}
		}

		fieldStr := ""
		if len(l.fields) > 0 {
			fieldStr = fmt.Sprintf(" fields: %v", l.fields)
		}
		logLine := fmt.Sprintf("[JORM] %s %s: %s%s\n", now.Format("2006-01-02 15:04:05"), level, msg, fieldStr)
		for _, w := range writers {
			// Don't use color for non-terminal outputs if possible, but for simplicity we keep it here
			// A better implementation would check if w is a terminal
			w.Write([]byte(logLine))
		}
	}
}

func (l *stdLogger) parseLevel(level string) LogLevel {
	switch strings.ToUpper(level) {
	case "ERROR":
		return LogLevelError
	case "WARN":
		return LogLevelWarn
	case "INFO", "SQL":
		return LogLevelInfo
	default:
		return LogLevelInfo
	}
}

func getSQLColor(sqlStr string) string {
	s := strings.TrimSpace(strings.ToUpper(sqlStr))
	switch {
	case strings.HasPrefix(s, "SELECT"):
		return ansiYellow
	case strings.HasPrefix(s, "INSERT"), strings.HasPrefix(s, "UPDATE"):
		return ansiGreen
	case strings.HasPrefix(s, "DELETE"):
		return ansiRed
	default:
		return ansiCyan
	}
}
