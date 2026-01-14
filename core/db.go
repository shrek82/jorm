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
		return nil, err
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
		return nil, err
	}

	return &DB{
		pool:    p,
		dialect: d,
		logger:  logger.NewStdLogger(),
	}, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.pool.Close()
}

// SetLogger sets a custom logger for the DB.
func (db *DB) SetLogger(l logger.Logger) {
	db.logger = l
}

// Model starts a new query builder for the given model instance.
func (db *DB) Model(value any) *Query {
	builder := NewBuilder(db.dialect)
	q := NewQuery(db, db.pool, builder)
	return q.Model(value)
}

// Table starts a new query builder for the given table name.
func (db *DB) Table(name string) *Query {
	builder := NewBuilder(db.dialect)
	q := NewQuery(db, db.pool, builder)
	return q.Table(name)
}

// Raw starts a new query with a raw SQL statement.
func (db *DB) Raw(sql string, args ...any) *Query {
	builder := NewBuilder(db.dialect)
	q := NewQuery(db, db.pool, builder)
	return q.Raw(sql, args...)
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
	return res, err
}

// Transaction executes a function within a database transaction.
func (db *DB) Transaction(fn func(tx *Tx) error) error {
	start := time.Now()
	sqlTx, err := db.pool.Begin()
	db.logSQL("BEGIN", time.Since(start))
	if err != nil {
		return err
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
		}
	}()

	err = fn(tx)
	return err
}

// AutoMigrate creates the table for the given model if it doesn't exist.
func (db *DB) AutoMigrate(values ...any) error {
	for _, value := range values {
		m, err := model.GetModel(value)
		if err != nil {
			return err
		}

		// Check if table exists
		sqlStr, args := db.dialect.HasTableSQL(m.TableName)
		var count int
		err = db.pool.QueryRowContext(context.Background(), sqlStr, args...).Scan(&count)
		if err != nil {
			return err
		}

		if count == 0 {
			// Create table
			createSQL, createArgs := db.dialect.CreateTableSQL(m)
			start := time.Now()
			_, err = db.pool.ExecContext(context.Background(), createSQL, createArgs...)
			db.logSQL(createSQL, time.Since(start), createArgs...)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
