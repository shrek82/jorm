package middleware

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/shrek82/jorm/core"
)

// SlowLogMiddleware logs queries that take longer than the specified threshold.
type SlowLogMiddleware struct {
	Threshold time.Duration
	LogPath   string
	logger    *log.Logger
	file      *os.File
}

// NewSlowLog creates a new SlowLogMiddleware.
// threshold: queries taking longer than this will be logged.
// logPath: path to the log file. If empty, logs to standard output.
func NewSlowLog(threshold time.Duration, logPath string) *SlowLogMiddleware {
	return &SlowLogMiddleware{
		Threshold: threshold,
		LogPath:   logPath,
	}
}

// SetOutput sets the output destination for the logger.
// This is useful for testing or custom logging.
func (m *SlowLogMiddleware) SetOutput(w io.Writer) {
	m.logger = log.New(w, "[SLOW SQL] ", log.LstdFlags)
}

func (m *SlowLogMiddleware) Name() string {
	return "SlowLog"
}

func (m *SlowLogMiddleware) Init(db *core.DB) error {
	// If logger is already set (e.g. by SetOutput), don't overwrite it
	if m.logger != nil {
		return nil
	}

	if m.LogPath != "" {
		f, err := os.OpenFile(m.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open slow log file: %w", err)
		}
		m.file = f
		m.logger = log.New(f, "[SLOW SQL] ", log.LstdFlags)
	} else {
		m.logger = log.New(os.Stdout, "[SLOW SQL] ", log.LstdFlags)
	}
	return nil
}

func (m *SlowLogMiddleware) Shutdown() error {
	if m.file != nil {
		return m.file.Close()
	}
	return nil
}

func (m *SlowLogMiddleware) Process(ctx context.Context, query *core.Query, next core.QueryFunc) (*core.Result, error) {
	start := time.Now()
	res, err := next(ctx, query)
	duration := time.Since(start)

	if duration > m.Threshold {
		sql := query.LastSQL
		args := query.LastArgs
		if sql == "" {
			// Try to get from result or query builder if not yet populated (though core should populate it)
			sql, args = query.GetSelectSQL()
		}
		m.logger.Printf("duration=%v | sql=%s | args=%v | rows=%d | err=%v", duration, sql, args, res.RowsAffected, res.Error)
	}

	return res, err
}
