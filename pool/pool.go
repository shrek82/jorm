package pool

import (
	"context"
	"database/sql"
	"time"
)

// Pool defines the interface for a database connection pool.
type Pool interface {
	Close() error
	SetMaxOpenConns(n int)
	SetMaxIdleConns(n int)
	SetConnMaxLifetime(d time.Duration)
	Ping() error
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	Begin() (*sql.Tx, error)
}

// StdPool is an implementation of Pool using the standard library's *sql.DB.
type StdPool struct {
	*sql.DB
}

// NewStdPool creates a new StdPool wrapping the given *sql.DB.
func NewStdPool(db *sql.DB) *StdPool {
	return &StdPool{db}
}
