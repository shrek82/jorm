package core

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/shrek82/jorm/model"
)

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
	preloads []*preloadConfig
}

type scanPlan struct {
	fields []*model.Field
}

type scanPlanKey struct {
	model *model.Model
	cols  string
}

var scanPlanCache sync.Map

func getScanPlan(m *model.Model, columns []string) *scanPlan {
	key := scanPlanKey{
		model: m,
		cols:  strings.Join(columns, ","),
	}
	if v, ok := scanPlanCache.Load(key); ok {
		return v.(*scanPlan)
	}

	fields := make([]*model.Field, len(columns))
	for i, col := range columns {
		// Try exact match first
		if field, ok := m.FieldMap[col]; ok {
			fields[i] = field
		} else {
			// Try matching with table prefix (e.g., "preload_user.name")
			parts := strings.Split(col, ".")
			if len(parts) > 1 {
				lastPart := parts[len(parts)-1]
				if field, ok := m.FieldMap[lastPart]; ok {
					fields[i] = field
				}
			}
		}
	}

	plan := &scanPlan{
		fields: fields,
	}
	scanPlanCache.Store(key, plan)
	return plan
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

func (q *Query) Alias(alias string) *Query {
	q.builder.Alias(alias)
	return q
}

func (q *Query) Select(columns ...string) *Query {
	q.builder.Select(columns...)
	return q
}

// Where adds a WHERE clause to the query.
func (q *Query) Where(cond string, args ...any) *Query {
	q.builder.Where(cond, args...)
	return q
}

func (q *Query) OrWhere(cond string, args ...any) *Query {
	q.builder.OrWhere(cond, args...)
	return q
}

func (q *Query) WhereIn(column string, values any) *Query {
	q.builder.WhereIn(column, values)
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

// Preload preloads the specified relation.
func (q *Query) Preload(name string) *Query {
	return q.PreloadWith(name, nil)
}

// PreloadWith preloads the specified relation with a custom query function.
func (q *Query) PreloadWith(name string, fn func(*Query)) *Query {
	path := strings.Split(name, ".")
	q.preloads = append(q.preloads, &preloadConfig{
		path:    path,
		builder: fn,
	})
	return q
}

// Joins adds a JOIN clause to the query.
// It supports raw SQL JOIN clauses: q.Joins("JOIN users ON users.id = orders.user_id")
func (q *Query) Joins(query string, args ...any) *Query {
	q.builder.Joins(query, args...)
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
	if err := q.queryRow(sqlStr, args, dest); err != nil {
		return err
	}
	return q.executePreloads(dest)
}

// Find retrieves all records matching the query into dest (must be a pointer to a slice).
func (q *Query) Find(dest any) error {
	defer PutBuilder(q.builder)
	if q.err != nil {
		return q.err
	}
	sqlStr, args := q.builder.BuildSelect()
	if err := q.queryRows(sqlStr, args, dest); err != nil {
		return err
	}
	return q.executePreloads(dest)
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
	q.db.logSQL(sqlStr, time.Since(start), args...)
	return count, err
}

func (q *Query) Sum(column string) (float64, error) {
	defer PutBuilder(q.builder)
	if q.err != nil {
		return 0, q.err
	}
	quoted := q.db.dialect.Quote(column)
	q.builder.Select("SUM(" + quoted + ")")
	sqlStr, args := q.builder.BuildSelect()

	var sum sql.NullFloat64
	start := time.Now()
	err := q.executor.QueryRowContext(q.ctx, sqlStr, args...).Scan(&sum)
	q.db.logSQL(sqlStr, time.Since(start), args...)
	if err != nil {
		return 0, err
	}
	if !sum.Valid {
		return 0, nil
	}
	return sum.Float64, nil
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
	q.db.logSQL(sqlStr, time.Since(start), args...)
	if err != nil {
		return fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return ErrRecordNotFound
	}

	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	m, err := model.GetModel(dest)
	if err != nil {
		return fmt.Errorf("failed to get model metadata: %w", err)
	}

	plan := getScanPlan(m, columns)

	if err := q.scanRowWithPlan(rows, dest, plan); err != nil {
		return fmt.Errorf("row scan failed: %w", err)
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
	q.db.logSQL(sqlStr, time.Since(start), args...)
	if err != nil {
		return fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to a slice")
	}

	sliceValue := destValue.Elem()
	itemType := sliceValue.Type().Elem()

	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	var m *model.Model
	var plan *scanPlan

	for rows.Next() {
		item := reflect.New(itemType).Interface()

		if plan == nil {
			m, err = model.GetModel(item)
			if err != nil {
				return fmt.Errorf("failed to get model metadata: %w", err)
			}
			plan = getScanPlan(m, columns)
		}

		if err := q.scanRowWithPlan(rows, item, plan); err != nil {
			return fmt.Errorf("row scan failed: %w", err)
		}

		// AfterFind hook
		if h, ok := item.(AfterFinder); ok {
			if err := h.AfterFind(); err != nil {
				return fmt.Errorf("AfterFind hook failed: %w", err)
			}
		}

		sliceValue.Set(reflect.Append(sliceValue, reflect.ValueOf(item).Elem()))
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows iteration error: %w", err)
	}
	return nil
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

	plan := getScanPlan(m, columns)
	return q.scanRowWithPlan(rows, dest, plan)
}

func (q *Query) scanRowWithPlan(rows *sql.Rows, dest any, plan *scanPlan) error {
	values := make([]any, len(plan.fields))
	for i, field := range plan.fields {
		if field != nil {
			// Ensure we are using the correct type for the scanner
			values[i] = reflect.New(field.Type).Interface()
		} else {
			var ignore any
			values[i] = &ignore
		}
	}

	if err := rows.Scan(values...); err != nil {
		return err
	}

	destValue := reflect.ValueOf(dest)
	if destValue.Kind() == reflect.Ptr {
		destValue = destValue.Elem()
	}

	for i, field := range plan.fields {
		if field != nil {
			val := reflect.ValueOf(values[i]).Elem()
			setFieldValue(destValue, field, val)
		}
	}

	return nil
}

func setFieldValue(dest reflect.Value, field *model.Field, value reflect.Value) {
	// Debug print
	// fmt.Printf("Setting field %s (type %v) with value %v (type %v)\n", field.Name, field.Type, value.Interface(), value.Type())

	if len(field.NestedIdx) > 0 {
		f := dest
		for _, i := range field.NestedIdx {
			if f.Kind() == reflect.Ptr {
				if f.IsNil() {
					if !f.CanSet() {
						return
					}
					f.Set(reflect.New(f.Type().Elem()))
				}
				f = f.Elem()
			}
			f = f.Field(i)
		}
		if f.CanSet() {
			// Handle type mismatch for basic types (e.g., int64 vs int)
			if f.Type() != value.Type() && value.Type().ConvertibleTo(f.Type()) {
				f.Set(value.Convert(f.Type()))
			} else if f.Type() == value.Type() {
				f.Set(value)
			}
		}
	} else {
		f := dest.Field(field.Index)
		if f.CanSet() {
			// Handle type mismatch for basic types
			if f.Type() != value.Type() && value.Type().ConvertibleTo(f.Type()) {
				f.Set(value.Convert(f.Type()))
			} else if f.Type() == value.Type() {
				f.Set(value)
			}
		}
	}
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
	q.db.logSQL(sqlStr, time.Since(start), args...)
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
	q.db.logSQL(sqlStr, time.Since(start), args...)
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
				// Only include non-zero values for struct updates
				if !fVal.IsZero() {
					data[field.Column] = fVal.Interface()
				}
			}
		}
	}

	sqlStr, args := q.builder.BuildUpdate(data)
	start := time.Now()
	res, err := q.executor.ExecContext(q.ctx, sqlStr, args...)
	q.db.logSQL(sqlStr, time.Since(start), args...)
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
	q.db.logSQL(sqlStr, time.Since(start), args...)
	if err != nil {
		return 0, err
	}

	return res.RowsAffected()
}
