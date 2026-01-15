package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/shrek82/jorm/dialect"
	"github.com/shrek82/jorm/logger"
	"github.com/shrek82/jorm/model"
	"github.com/shrek82/jorm/pool"
)

// Options defines the configuration for the DB connection pool and its behavior.
type Options struct {
	// MaxOpenConns sets the maximum number of open connections to the database.
	MaxOpenConns int
	// MaxIdleConns sets the maximum number of connections in the idle connection pool.
	MaxIdleConns int
	// ConnMaxLifetime sets the maximum amount of time a connection may be reused.
	ConnMaxLifetime time.Duration
	// MaxRetries specifies the maximum number of retry attempts for the initial connection.
	MaxRetries int
	// RetryDelay defines the initial duration to wait between connection retry attempts.
	RetryDelay time.Duration
}

// DB is the central engine of the JORM ORM.
// It manages the underlying connection pool, SQL dialect, and logging capabilities.
// Use core.Open to initialize a new instance.
type DB struct {
	pool    pool.Pool
	dialect dialect.Dialect
	logger  logger.Logger

	// Health tracking
	mu           sync.RWMutex
	lastErr      error
	lastErrTime  time.Time
	cooldownTime time.Duration
}

// Open initializes a new DB instance with the given driver and DSN.
// It sets up the dialect based on the driver and initializes the connection pool.
// The opts parameter can be used to configure connection pool settings like MaxOpenConns.
// If opts is nil, default settings are used.
func Open(driver, dsn string, opts *Options) (*DB, error) {
	d, ok := dialect.Get(driver)
	if !ok {
		return nil, fmt.Errorf("unknown dialect %s", driver)
	}

	sqlDB, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	p := pool.NewStdPool(sqlDB)

	maxRetries := 0
	retryDelay := time.Second
	if opts != nil {
		if opts.MaxOpenConns > 0 {
			p.SetMaxOpenConns(opts.MaxOpenConns)
		}
		if opts.MaxIdleConns > 0 {
			p.SetMaxIdleConns(opts.MaxIdleConns)
		}
		if opts.ConnMaxLifetime > 0 {
			p.SetConnMaxLifetime(opts.ConnMaxLifetime)
		}
		maxRetries = opts.MaxRetries
		if opts.RetryDelay > 0 {
			retryDelay = opts.RetryDelay
		}
	}

	var pingErr error
	for i := 0; i <= maxRetries; i++ {
		pingErr = p.Ping()
		if pingErr == nil {
			break
		}

		if i < maxRetries {
			// Exponential backoff: delay * 2^i
			actualDelay := retryDelay * (1 << uint(i))
			// Cap the delay to a reasonable maximum (e.g., 30 seconds) to avoid resource exhaustion
			if actualDelay > 30*time.Second {
				actualDelay = 30 * time.Second
			}
			time.Sleep(actualDelay)
		}
	}

	if pingErr != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("database ping failed after %d retries: %w", maxRetries, pingErr)
	}

	return &DB{
		pool:         p,
		dialect:      d,
		logger:       logger.NewStdLogger(),
		cooldownTime: 5 * time.Second, // Default cooldown if DB is down
	}, nil
}

// Close closes the database connection and releases any resources.
// It should be called when the DB instance is no longer needed.
func (db *DB) Close() error {
	if err := db.pool.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}
	return nil
}

// SetLogger sets a custom logger for the DB instance.
// The logger will be used to record SQL queries, execution times, and errors.
func (db *DB) SetLogger(l logger.Logger) {
	db.logger = l
}

// checkHealth verifies if the database connection is currently in a cooldown period
// due to recent connection failures.
func (db *DB) checkHealth() error {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.lastErr != nil && time.Since(db.lastErrTime) < db.cooldownTime {
		return fmt.Errorf("%w: in cooldown period until %v", ErrConnectionFailed, db.lastErrTime.Add(db.cooldownTime))
	}
	return nil
}

// reportError records a database error and triggers a cooldown period if the error
// is connection-related. Providing nil clears the error state.
func (db *DB) reportError(err error) {
	if err == nil {
		db.mu.Lock()
		db.lastErr = nil
		db.mu.Unlock()
		return
	}

	// Only trigger cooldown for connection-related errors
	// This is a simplified check; in a real-world scenario, you might want to check for specific network errors
	errMsg := err.Error()
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "i/o timeout") ||
		strings.Contains(errMsg, "reset by peer") ||
		strings.Contains(errMsg, "broken pipe") ||
		errors.Is(err, ErrConnectionFailed) {

		db.mu.Lock()
		db.lastErr = err
		db.lastErrTime = time.Now()
		db.mu.Unlock()
	}
}

// newQuery creates a new Query instance associated with this DB.
// It initializes the query builder and checks for database health.
func (db *DB) newQuery(executor Executor) *Query {
	builder := NewBuilder(db.dialect)
	q := NewQuery(db, executor, builder)
	if err := db.checkHealth(); err != nil {
		q.err = err
	}
	return q
}

// Model starts a new query builder for the given model instance.
// The value parameter can be a struct pointer or a slice of struct pointers.
// JORM will automatically detect the table name and fields from the model.
func (db *DB) Model(value any) *Query {
	return db.newQuery(db.pool).Model(value)
}

// Table starts a new query builder for the given table name.
// This is useful for performing operations on tables that don't have a corresponding model struct.
func (db *DB) Table(name string) *Query {
	return db.newQuery(db.pool).Table(name)
}

// Raw starts a new query with a raw SQL statement and its arguments.
// It bypasses the JORM query builder and allows for direct SQL execution.
func (db *DB) Raw(sql string, args ...any) *Query {
	return db.newQuery(db.pool).Raw(sql, args...)
}

// logSQL logs the SQL statement, its execution duration, and arguments.
// It only logs if a logger has been configured for the DB.
func (db *DB) logSQL(sql string, duration time.Duration, args ...any) {
	if db.logger != nil {
		db.logger.SQL(sql, duration, args...)
	}
}

// Exec executes a raw SQL statement without returning any rows.
// It returns a sql.Result and any error encountered during execution.
// It also handles health checks and error reporting.
func (db *DB) Exec(sql string, args ...any) (sql.Result, error) {
	if err := db.checkHealth(); err != nil {
		return nil, err
	}
	start := time.Now()
	res, err := db.pool.ExecContext(context.Background(), sql, args...)
	db.logSQL(sql, time.Since(start), args...)
	if err != nil {
		db.reportError(err)
		return nil, fmt.Errorf("failed to execute sql [%s]: %w", sql, err)
	}
	db.reportError(nil)
	return res, nil
}

// Transaction executes the provided function within a database transaction.
// If the function returns an error or panics, the transaction is automatically rolled back.
// Otherwise, the transaction is committed.
func (db *DB) Transaction(fn func(tx *Tx) error) (err error) {
	if err := db.checkHealth(); err != nil {
		return err
	}
	start := time.Now()
	sqlTx, err := db.pool.Begin()
	db.logSQL("BEGIN", time.Since(start))
	if err != nil {
		db.reportError(err)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	db.reportError(nil)

	tx := &Tx{
		db:    db,
		sqlTx: sqlTx,
	}

	defer func() {
		if p := recover(); p != nil {
			start := time.Now()
			_ = sqlTx.Rollback()
			db.logSQL("ROLLBACK", time.Since(start))
			panic(p)
		} else if err != nil {
			start := time.Now()
			_ = sqlTx.Rollback()
			db.logSQL("ROLLBACK", time.Since(start))
		} else {
			start := time.Now()
			err = sqlTx.Commit()
			db.logSQL("COMMIT", time.Since(start))
			if err != nil {
				err = fmt.Errorf("failed to commit transaction: %w", err)
			}
		}
	}()

	err = fn(tx)
	return err
}

// HasTable checks if the specified table exists in the database.
// It uses the dialect-specific implementation to perform the check.
func (db *DB) HasTable(tableName string) (bool, error) {
	sqlStr, args := db.dialect.HasTableSQL(tableName)
	var count int
	err := db.pool.QueryRowContext(context.Background(), sqlStr, args...).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if table %s exists: %w", tableName, err)
	}
	return count > 0, nil
}

// AutoMigrate creates or updates the table for the given model.
func (db *DB) AutoMigrate(values ...any) error {
	for _, value := range values {
		m, err := model.GetModel(value)
		if err != nil {
			return fmt.Errorf("failed to get model for migration: %w", err)
		}

		exists, err := db.HasTable(m.TableName)
		if err != nil {
			return err
		}

		if !exists {
			createSQL, createArgs := db.dialect.CreateTableSQL(m)
			_, err = db.Exec(createSQL, createArgs...)
			if err != nil {
				return fmt.Errorf("failed to create table %s: %w", m.TableName, err)
			}
		} else {
			if err := db.alterTableIfNeeded(m); err != nil {
				return err
			}
		}

		if err := db.syncIndexes(m); err != nil {
			return err
		}
	}
	return nil
}

// alterTableIfNeeded compares the model definition with the existing table schema
// and adds any missing columns.
func (db *DB) alterTableIfNeeded(m *model.Model) error {
	sqlStr, args := db.dialect.GetColumnsSQL(m.TableName)
	rows, err := db.pool.QueryContext(context.Background(), sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to get columns for table %s: %w", m.TableName, err)
	}
	defer rows.Close()

	colNames, err := db.dialect.ParseColumns(rows)
	if err != nil {
		return fmt.Errorf("failed to parse columns for table %s: %w", m.TableName, err)
	}

	existingColumns := make(map[string]bool)
	for _, name := range colNames {
		existingColumns[name] = true
	}

	for _, field := range m.Fields {
		if !existingColumns[field.Column] {
			// Add missing column
			addSql, addArgs := db.dialect.AddColumnSQL(m.TableName, field)
			if addSql != "" {
				_, err = db.Exec(addSql, addArgs...)
				if err != nil {
					return fmt.Errorf("failed to add column %s to table %s: %w", field.Column, m.TableName, err)
				}
			}
		}
	}

	return nil
}

func (db *DB) syncIndexes(m *model.Model) error {
	sqlStr, args := db.dialect.GetIndexesSQL(m.TableName)
	rows, err := db.pool.QueryContext(context.Background(), sqlStr, args...)
	if err != nil {
		return fmt.Errorf("failed to get indexes for table %s: %w", m.TableName, err)
	}
	defer rows.Close()

	existingIndexes, err := db.dialect.ParseIndexes(rows)
	if err != nil {
		return fmt.Errorf("failed to parse indexes for table %s: %w", m.TableName, err)
	}

	// Helper to check if an index exists with the same columns
	hasIndex := func(columns []string, unique bool) bool {
		for _, existingCols := range existingIndexes {
			if len(existingCols) == 0 {
				// For SQLite, ParseIndexes might return index names with empty columns for now
				// This is a limitation we might need to fix, but for now we'll match by name convention if columns are empty
				continue
			}
			if len(existingCols) != len(columns) {
				continue
			}
			match := true
			for i, col := range columns {
				if strings.ToLower(existingCols[i]) != strings.ToLower(col) {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
		return false
	}

	for _, field := range m.Fields {
		if field.IsUnique {
			indexName := fmt.Sprintf("idx_%s_%s", m.TableName, field.Column)

			// For SQLite, check by name if columns are empty
			existsByName := false
			if _, ok := existingIndexes[indexName]; ok {
				existsByName = true
			}

			if !existsByName && !hasIndex([]string{field.Column}, true) {
				createIdxSQL, createIdxArgs := db.dialect.CreateIndexSQL(m.TableName, indexName, []string{field.Column}, true)
				if createIdxSQL != "" {
					_, err = db.Exec(createIdxSQL, createIdxArgs...)
					if err != nil {
						return fmt.Errorf("failed to create unique index %s on table %s: %w", indexName, m.TableName, err)
					}
				}
			}
		}
	}

	return nil
}
