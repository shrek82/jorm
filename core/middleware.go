package core

import (
	"context"
	"database/sql"
)

// Component is the base interface for all JORM components/middleware.
type Component interface {
	Name() string
	Init(db *DB) error
	Shutdown() error
}

// Result represents the result of a query execution.
type Result struct {
	RowsAffected int64
	LastInsertId int64
	Data         any // The destination (pointer to slice or struct)
	Error        error
	RawRows      *sql.Rows // For advanced usage
}

// QueryFunc is the function type for the next step in the middleware chain.
type QueryFunc func(ctx context.Context, query *Query) (*Result, error)

// QueryMiddleware is the interface for query interceptors.
type QueryMiddleware interface {
	Component
	Process(ctx context.Context, query *Query, next QueryFunc) (*Result, error)
}
