package core

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"time"

	"github.com/shrek82/jorm/model"
)

// Hook interfaces for model lifecycle events
type BeforeInserter interface{ BeforeInsert() error }
type AfterInserter interface{ AfterInsert(id int64) error }
type BeforeUpdater interface{ BeforeUpdate() error }
type AfterUpdater interface{ AfterUpdate() error }
type BeforeDeleter interface{ BeforeDelete() error }
type AfterDeleter interface{ AfterDelete() error }
type AfterFinder interface{ AfterFind() error }

// Executor defines the interface for executing SQL queries and commands.
// It is implemented by *sql.DB and *sql.Tx.
type Executor interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Query is the chainable query builder and executor.
type Query struct {
	db       *DB
	executor Executor
	builder  Builder
	ctx      context.Context
	model    *model.Model
	err      error
	rawSQL   string
	rawArgs  []any
}

// NewQuery creates a new Query instance.
func NewQuery(db *DB, executor Executor, builder Builder) *Query {
	return &Query{
		db:       db,
		executor: executor,
		builder:  builder,
		ctx:      context.Background(),
	}
}

// Model sets the target model for the query and parses its metadata.
func (q *Query) Model(value any) *Query {
	m, err := model.GetModel(value)
	if err != nil {
		q.err = err
		return q
	}
	q.model = m
	q.builder.SetTable(m.TableName)
	return q
}

// Table sets the target table name for the query.
func (q *Query) Table(name string) *Query {
	q.builder.SetTable(name)
	return q
}

// Where adds a WHERE clause to the query.
func (q *Query) Where(cond string, args ...any) *Query {
	q.builder.Where(cond, args...)
	return q
}

// Limit sets the LIMIT clause.
func (q *Query) Limit(n int) *Query {
	q.builder.Limit(n)
	return q
}

// Offset sets the OFFSET clause.
func (q *Query) Offset(n int) *Query {
	q.builder.Offset(n)
	return q
}

// OrderBy adds an ORDER BY clause.
func (q *Query) OrderBy(columns ...string) *Query {
	q.builder.OrderBy(columns...)
	return q
}

// WithContext sets the context for the query execution.
func (q *Query) WithContext(ctx context.Context) *Query {
	q.ctx = ctx
	return q
}

// Raw sets a raw SQL query and its arguments.
func (q *Query) Raw(sql string, args ...any) *Query {
	q.rawSQL = sql
	q.rawArgs = args
	return q
}

// First retrieves the first record matching the query into dest.
func (q *Query) First(dest any) error {
	defer PutBuilder(q.builder)
	if q.err != nil {
		return q.err
	}
	q.builder.Limit(1)
	sqlStr, args := q.builder.BuildSelect()
	return q.queryRow(sqlStr, args, dest)
}

// Find retrieves all records matching the query into dest (must be a pointer to a slice).
func (q *Query) Find(dest any) error {
	defer PutBuilder(q.builder)
	if q.err != nil {
		return q.err
	}
	sqlStr, args := q.builder.BuildSelect()
	return q.queryRows(sqlStr, args, dest)
}

// Count returns the number of records matching the query.
func (q *Query) Count() (int64, error) {
	defer PutBuilder(q.builder)
	if q.err != nil {
		return 0, q.err
	}
	q.builder.Select("COUNT(*)")
	sqlStr, args := q.builder.BuildSelect()

	var count int64
	start := time.Now()
	err := q.executor.QueryRowContext(q.ctx, sqlStr, args...).Scan(&count)
	q.db.logger.SQL(sqlStr, time.Since(start), args...)
	return count, err
}

// Scan executes a raw query and scans the result into dest.
func (q *Query) Scan(dest any) error {
	if q.rawSQL == "" {
		return fmt.Errorf("raw sql is empty")
	}
	return q.queryRows(q.rawSQL, q.rawArgs, dest)
}

func (q *Query) queryRow(sqlStr string, args []any, dest any) error {
	start := time.Now()
	rows, err := q.executor.QueryContext(q.ctx, sqlStr, args...)
	q.db.logger.SQL(sqlStr, time.Since(start), args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		return ErrRecordNotFound
	}

	if err := q.scanRow(rows, dest); err != nil {
		return err
	}

	// AfterFind hook
	if h, ok := dest.(AfterFinder); ok {
		return h.AfterFind()
	}
	return nil
}

func (q *Query) queryRows(sqlStr string, args []any, dest any) error {
	start := time.Now()
	rows, err := q.executor.QueryContext(q.ctx, sqlStr, args...)
	q.db.logger.SQL(sqlStr, time.Since(start), args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to a slice")
	}

	sliceValue := destValue.Elem()
	itemType := sliceValue.Type().Elem()

	for rows.Next() {
		item := reflect.New(itemType).Interface()
		if err := q.scanRow(rows, item); err != nil {
			return err
		}

		// AfterFind hook
		if h, ok := item.(AfterFinder); ok {
			if err := h.AfterFind(); err != nil {
				return err
			}
		}

		sliceValue.Set(reflect.Append(sliceValue, reflect.ValueOf(item).Elem()))
	}

	return rows.Err()
}

func (q *Query) scanRow(rows *sql.Rows, dest any) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	m, err := model.GetModel(dest)
	if err != nil {
		return err
	}

	values := make([]any, len(columns))
	for i, col := range columns {
		if field, ok := m.FieldMap[col]; ok {
			values[i] = reflect.New(field.Type).Interface()
		} else {
			var ignore any
			values[i] = &ignore
		}
	}

	if err := rows.Scan(values...); err != nil {
		return err
	}

	destValue := reflect.ValueOf(dest).Elem()
	for i, col := range columns {
		if field, ok := m.FieldMap[col]; ok {
			destValue.Field(field.Index).Set(reflect.ValueOf(values[i]).Elem())
		}
	}

	return nil
}

// Insert inserts a new record into the database.
func (q *Query) Insert(value any) (int64, error) {
	defer PutBuilder(q.builder)
	if q.err != nil {
		return 0, q.err
	}
	m, err := model.GetModel(value)
	if err != nil {
		return 0, err
	}

	// BeforeInsert hook
	if h, ok := value.(BeforeInserter); ok {
		if err := h.BeforeInsert(); err != nil {
			return 0, err
		}
	}

	var columns []string
	var args []any
	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	now := time.Now()
	for _, field := range m.Fields {
		if field.IsAuto {
			continue
		}
		fVal := val.Field(field.Index)
		if (field.AutoTime || field.AutoUpdate) && fVal.CanSet() {
			fVal.Set(reflect.ValueOf(now))
		}
		columns = append(columns, field.Column)
		args = append(args, fVal.Interface())
	}

	sqlStr, _ := q.builder.BuildInsert(columns)
	start := time.Now()
	res, err := q.executor.ExecContext(q.ctx, sqlStr, args...)
	q.db.logger.SQL(sqlStr, time.Since(start), args...)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	// AfterInsert hook
	if h, ok := value.(AfterInserter); ok {
		if err := h.AfterInsert(id); err != nil {
			return id, err
		}
	}

	return id, nil
}

// BatchInsert inserts multiple records into the database.
// values must be a slice of structs or pointers to structs.
func (q *Query) BatchInsert(values any) (int64, error) {
	defer PutBuilder(q.builder)
	if q.err != nil {
		return 0, q.err
	}

	sliceVal := reflect.ValueOf(values)
	if sliceVal.Kind() != reflect.Slice {
		return 0, fmt.Errorf("values must be a slice")
	}

	if sliceVal.Len() == 0 {
		return 0, nil
	}

	// Use the first element to get model info
	m, err := model.GetModel(sliceVal.Index(0).Interface())
	if err != nil {
		return 0, err
	}

	var columns []string
	for _, field := range m.Fields {
		if !field.IsAuto {
			columns = append(columns, field.Column)
		}
	}

	sqlStr, _ := q.db.dialect.BatchInsertSQL(m.TableName, columns, sliceVal.Len())
	var args []any
	now := time.Now()

	for i := 0; i < sliceVal.Len(); i++ {
		item := sliceVal.Index(i).Interface()
		val := reflect.ValueOf(item)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}

		// Hooks
		if h, ok := item.(BeforeInserter); ok {
			if err := h.BeforeInsert(); err != nil {
				return 0, err
			}
		}

		for _, field := range m.Fields {
			if field.IsAuto {
				continue
			}
			fVal := val.Field(field.Index)
			if (field.AutoTime || field.AutoUpdate) && fVal.CanSet() {
				fVal.Set(reflect.ValueOf(now))
			}
			args = append(args, fVal.Interface())
		}
	}

	start := time.Now()
	res, err := q.executor.ExecContext(q.ctx, sqlStr, args...)
	q.db.logger.SQL(sqlStr, time.Since(start), args...)
	if err != nil {
		return 0, err
	}

	totalAffected, _ := res.RowsAffected()

	// AfterInsert hooks (Batch)
	for i := 0; i < sliceVal.Len(); i++ {
		item := sliceVal.Index(i).Interface()
		if h, ok := item.(AfterInserter); ok {
			// Note: LastInsertId in batch mode is driver-dependent
			// Usually returns the first ID of the batch
			id, _ := res.LastInsertId()
			if err := h.AfterInsert(id + int64(i)); err != nil {
				return totalAffected, err
			}
		}
	}

	return totalAffected, nil
}

// Update updates records matching the query.
// value can be a map[string]any or a struct.
func (q *Query) Update(value any) (int64, error) {
	defer PutBuilder(q.builder)
	if q.err != nil {
		return 0, q.err
	}

	// BeforeUpdate hook
	if h, ok := value.(BeforeUpdater); ok {
		if err := h.BeforeUpdate(); err != nil {
			return 0, err
		}
	}

	var data map[string]any
	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Map {
		data = value.(map[string]any)
	} else {
		// Convert struct to map
		m, err := model.GetModel(value)
		if err != nil {
			return 0, err
		}
		data = make(map[string]any)
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		now := time.Now()
		for _, field := range m.Fields {
			if !field.IsPK && !field.IsAuto {
				fVal := val.Field(field.Index)
				if field.AutoUpdate && fVal.CanSet() {
					fVal.Set(reflect.ValueOf(now))
				}
				data[field.Column] = fVal.Interface()
			}
		}
	}

	sqlStr, args := q.builder.BuildUpdate(data)
	start := time.Now()
	res, err := q.executor.ExecContext(q.ctx, sqlStr, args...)
	q.db.logger.SQL(sqlStr, time.Since(start), args...)
	if err != nil {
		return 0, err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	// AfterUpdate hook
	if h, ok := value.(AfterUpdater); ok {
		if err := h.AfterUpdate(); err != nil {
			return affected, err
		}
	}

	return affected, nil
}

// Delete deletes records matching the query.
func (q *Query) Delete() (int64, error) {
	defer PutBuilder(q.builder)
	if q.err != nil {
		return 0, q.err
	}

	sqlStr, args := q.builder.BuildDelete()
	start := time.Now()
	res, err := q.executor.ExecContext(q.ctx, sqlStr, args...)
	q.db.logger.SQL(sqlStr, time.Since(start), args...)
	if err != nil {
		return 0, err
	}

	return res.RowsAffected()
}
