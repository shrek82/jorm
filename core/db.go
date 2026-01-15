package core

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/shrek82/jorm/dialect"
	"github.com/shrek82/jorm/logger"
	"github.com/shrek82/jorm/model"
	"github.com/shrek82/jorm/pool"
)

// Options defines the configuration for the DB connection pool.
type Options struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// DB is the main entry point for the ORM.
// It manages the database connection pool and provides methods to create queries.
type DB struct {
	pool    pool.Pool
	dialect dialect.Dialect
	logger  logger.Logger
}

// Open initializes a new DB instance with the given driver and DSN.
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
	}

	if err := p.Ping(); err != nil {
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	return &DB{
		pool:    p,
		dialect: d,
		logger:  logger.NewStdLogger(),
	}, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	if err := db.pool.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}
	return nil
}

// SetLogger sets a custom logger for the DB.
func (db *DB) SetLogger(l logger.Logger) {
	db.logger = l
}

func (db *DB) newQuery(executor Executor) *Query {
	builder := NewBuilder(db.dialect)
	return NewQuery(db, executor, builder)
}

// Model starts a new query builder for the given model instance.
func (db *DB) Model(value any) *Query {
	return db.newQuery(db.pool).Model(value)
}

// Table starts a new query builder for the given table name.
func (db *DB) Table(name string) *Query {
	return db.newQuery(db.pool).Table(name)
}

// Raw starts a new query with a raw SQL statement.
func (db *DB) Raw(sql string, args ...any) *Query {
	return db.newQuery(db.pool).Raw(sql, args...)
}

// logSQL logs the SQL execution if a logger is set.
func (db *DB) logSQL(sql string, duration time.Duration, args ...any) {
	if db.logger != nil {
		db.logger.SQL(sql, duration, args...)
	}
}

// Exec executes a raw SQL statement without returning any rows.
func (db *DB) Exec(sql string, args ...any) (sql.Result, error) {
	start := time.Now()
	res, err := db.pool.ExecContext(context.Background(), sql, args...)
	db.logSQL(sql, time.Since(start), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute sql [%s]: %w", sql, err)
	}
	return res, nil
}

// Transaction executes a function within a database transaction.
func (db *DB) Transaction(fn func(tx *Tx) error) (err error) {
	start := time.Now()
	sqlTx, err := db.pool.Begin()
	db.logSQL("BEGIN", time.Since(start))
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

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

// HasTable checks if a table exists in the database.
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
	// TODO: Implement index sync
	return nil
}
