package core

import (
	"context"
	"database/sql"
	"fmt"
)

// Tx represents a database transaction.
// It implements the Executor interface and provides methods to create queries within the transaction.
type Tx struct {
	db    *DB
	sqlTx *sql.Tx
}

// Model starts a new query builder for the given model instance within the transaction.
func (tx *Tx) Model(value any) *Query {
	return tx.db.newQuery(tx).Model(value)
}

// Table starts a new query builder for the given table name within the transaction.
func (tx *Tx) Table(name string) *Query {
	return tx.db.newQuery(tx).Table(name)
}

// Commit commits the transaction.
func (tx *Tx) Commit() error {
	if err := tx.sqlTx.Commit(); err != nil {
		return fmt.Errorf("transaction commit failed: %w", err)
	}
	return nil
}

// Rollback rolls back the transaction.
func (tx *Tx) Rollback() error {
	if err := tx.sqlTx.Rollback(); err != nil {
		return fmt.Errorf("transaction rollback failed: %w", err)
	}
	return nil
}

// QueryContext executes a query that returns rows, typically a SELECT.
func (tx *Tx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	rows, err := tx.sqlTx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("transaction query failed: %w", err)
	}
	return rows, nil
}

// QueryRowContext executes a query that is expected to return at most one row.
func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return tx.sqlTx.QueryRowContext(ctx, query, args...)
}

// ExecContext executes a query that doesn't return rows, such as an INSERT or UPDATE.
func (tx *Tx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	res, err := tx.sqlTx.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("transaction exec failed: %w", err)
	}
	return res, nil
}
