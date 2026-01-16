package core

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/shrek82/jorm/logger"
	"github.com/shrek82/jorm/model"
	"github.com/shrek82/jorm/validator"
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
	logger   logger.Logger
}

type scanPlan struct {
	fields     []*model.Field
	converters []converter
}

type converter func(src, dst reflect.Value)

var converterCache sync.Map

func getConverter(srcType, dstType reflect.Type) converter {
	key := srcType.String() + "->" + dstType.String()
	if v, ok := converterCache.Load(key); ok {
		return v.(converter)
	}

	var conv converter
	if srcType == dstType {
		conv = func(src, dst reflect.Value) {
			dst.Set(src)
		}
	} else if srcType.ConvertibleTo(dstType) {
		conv = func(src, dst reflect.Value) {
			dst.Set(src.Convert(dstType))
		}
	} else {
		conv = func(src, dst reflect.Value) {
			// Do nothing or handle error? The original code ignored failures.
		}
	}

	converterCache.Store(key, conv)
	return conv
}

type scanBuffer struct {
	values []any
}

var scanBufferPool = sync.Pool{
	New: func() any {
		return &scanBuffer{
			values: make([]any, 0, 32),
		}
	},
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
	converters := make([]converter, len(columns))

	for i, col := range columns {
		// Try exact match first
		var field *model.Field
		if f, ok := m.FieldMap[col]; ok {
			field = f
		} else {
			// Try matching with table prefix (e.g., "preload_user.name")
			parts := strings.Split(col, ".")
			if len(parts) > 1 {
				lastPart := parts[len(parts)-1]
				if f, ok := m.FieldMap[lastPart]; ok {
					field = f
				}
			}
		}

		if field != nil {
			fields[i] = field
			// We can't easily get the destination field type here because of NestedIdx
			// Let's keep it simple for now and cache the converter in setFieldValue if needed
		}
	}

	plan := &scanPlan{
		fields:     fields,
		converters: converters,
	}
	scanPlanCache.Store(key, plan)
	return plan
}

// NewQuery creates a new Query instance with the specified DB, executor, and builder.
// This is typically called internally by DB.Model, DB.Table, or DB.Raw.
func NewQuery(db *DB, executor Executor, builder Builder) *Query {
	return &Query{
		db:       db,
		executor: executor,
		builder:  builder,
		ctx:      context.Background(),
		logger:   db.logger,
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

// Select specifies the columns to be retrieved by the query.
// If not called, all columns (*) will be selected by default.
func (q *Query) Select(columns ...string) *Query {
	q.builder.Select(columns...)
	return q
}

// Where adds a WHERE clause to the query.
func (q *Query) Where(cond string, args ...any) *Query {
	q.builder.Where(cond, args...)
	return q
}

// OrWhere adds an OR condition to the WHERE clause of the query.
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

// WithFields adds structured fields to the query's logger.
func (q *Query) WithFields(fields map[string]any) *Query {
	if q.logger != nil {
		q.logger = q.logger.WithFields(fields)
	}
	return q
}

func (q *Query) logSQL(sql string, duration time.Duration, args ...any) {
	if q.logger != nil {
		q.logger.SQL(sql, duration, args...)
	} else if q.db != nil {
		q.db.logSQL(sql, duration, args...)
	}
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

// GroupBy adds a GROUP BY clause to the query for the specified columns.
func (q *Query) GroupBy(columns ...string) *Query {
	q.builder.GroupBy(columns...)
	return q
}

func (q *Query) Having(cond string, args ...any) *Query {
	q.builder.Having(cond, args...)
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
		return fmt.Errorf("First failed: %w", err)
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
		return fmt.Errorf("Find failed: %w", err)
	}
	return q.executePreloads(dest)
}

// Count returns the total number of records matching the query.
// It executes a "SELECT COUNT(*)" query and returns the result as an int64.
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
	q.logSQL(sqlStr, time.Since(start), args...)
	if err != nil {
		return 0, fmt.Errorf("Count failed: %w", err)
	}
	return count, nil
}

// Sum calculates the sum of the specified numeric column for records matching the query.
// It returns a float64 value and any error encountered.
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
	q.logSQL(sqlStr, time.Since(start), args...)
	if err != nil {
		return 0, fmt.Errorf("Sum failed for column %s: %w", column, err)
	}
	if !sum.Valid {
		return 0, nil
	}
	return sum.Float64, nil
}

// Scan executes a raw query and scans the result into dest.
// dest can be a pointer to a struct or a pointer to a slice.
func (q *Query) Scan(dest any) error {
	if q.rawSQL == "" {
		return fmt.Errorf("raw sql is empty")
	}

	val := reflect.ValueOf(dest)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer")
	}

	if val.Elem().Kind() == reflect.Slice {
		return q.queryRows(q.rawSQL, q.rawArgs, dest)
	}
	return q.queryRow(q.rawSQL, q.rawArgs, dest)
}

// Exec executes a raw SQL statement and returns the number of affected rows.
func (q *Query) Exec() (int64, error) {
	if q.rawSQL == "" {
		return 0, fmt.Errorf("raw sql is empty")
	}

	start := time.Now()
	res, err := q.executor.ExecContext(q.ctx, q.rawSQL, q.rawArgs...)
	q.logSQL(q.rawSQL, time.Since(start), q.rawArgs...)
	if err != nil {
		return 0, q.handleError(fmt.Errorf("raw sql execution failed: %w", err))
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return 0, q.handleError(fmt.Errorf("failed to get affected rows: %w", err))
	}

	return rows, nil
}

func (q *Query) handleError(err error) error {
	if err != nil && q.db != nil {
		q.db.reportError(err)
	}
	return err
}

func (q *Query) queryRow(sqlStr string, args []any, dest any) error {
	start := time.Now()
	rows, err := q.executor.QueryContext(q.ctx, sqlStr, args...)
	q.logSQL(sqlStr, time.Since(start), args...)
	if err != nil {
		return q.handleError(fmt.Errorf("query execution failed: %w", err))
	}
	defer rows.Close()

	if !rows.Next() {
		return ErrRecordNotFound
	}

	columns, err := rows.Columns()
	if err != nil {
		return q.handleError(fmt.Errorf("failed to get columns: %w", err))
	}

	m, err := model.GetModel(dest)
	if err != nil {
		return q.handleError(fmt.Errorf("failed to get model metadata: %w", err))
	}

	plan := getScanPlan(m, columns)

	if err := q.scanRowWithPlan(rows, dest, plan); err != nil {
		return q.handleError(fmt.Errorf("row scan failed: %w", err))
	}

	// AfterFind hook
	if m.HasAfterFind {
		if h, ok := dest.(model.AfterFinder); ok {
			if err := h.AfterFind(); err != nil {
				return q.handleError(fmt.Errorf("AfterFind hook failed: %w", err))
			}
		}
	}

	// Success! Clear cooldown if any
	q.handleError(nil)
	return nil
}

func (q *Query) queryRows(sqlStr string, args []any, dest any) error {
	start := time.Now()
	rows, err := q.executor.QueryContext(q.ctx, sqlStr, args...)
	q.logSQL(sqlStr, time.Since(start), args...)
	if err != nil {
		return q.handleError(fmt.Errorf("query execution failed: %w", err))
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
		return q.handleError(fmt.Errorf("failed to get columns: %w", err))
	}

	var m *model.Model
	var plan *scanPlan

	for rows.Next() {
		item := reflect.New(itemType)
		itemInterface := item.Interface()

		if plan == nil {
			m, err = model.GetModel(itemInterface)
			if err != nil {
				return q.handleError(fmt.Errorf("failed to get model metadata: %w", err))
			}
			plan = getScanPlan(m, columns)
		}

		// Pass reflect.Value directly to avoid repeated reflect.ValueOf
		if err := q.scanRowWithPlan(rows, item.Elem(), plan); err != nil {
			return q.handleError(fmt.Errorf("row scan failed: %w", err))
		}

		// AfterFind hook
		if m.HasAfterFind {
			if h, ok := itemInterface.(model.AfterFinder); ok {
				if err := h.AfterFind(); err != nil {
					return fmt.Errorf("AfterFind hook failed: %w", err)
				}
			}
		}

		sliceValue.Set(reflect.Append(sliceValue, item.Elem()))
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows iteration error: %w", err)
	}
	return nil
}

func (q *Query) scanRow(rows *sql.Rows, dest any) error {
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	m, err := model.GetModel(dest)
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}

	plan := getScanPlan(m, columns)
	if err := q.scanRowWithPlan(rows, dest, plan); err != nil {
		return fmt.Errorf("failed to scan row with plan: %w", err)
	}
	return nil
}

var (
	timeType    = reflect.TypeOf(time.Time{})
	timePtrType = reflect.TypeOf((*time.Time)(nil))
)

// TimeScanner acts as a scanner for time.Time fields.
// It handles various formats including strings, bytes, and native time.Time.
type TimeScanner struct {
	Value time.Time
	Valid bool
}

// Scan implements the sql.Scanner interface.
func (s *TimeScanner) Scan(value any) error {
	if value == nil {
		s.Valid = false
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		s.Value = v
		s.Valid = true
		return nil
	case []byte:
		return s.parse(string(v))
	case string:
		return s.parse(v)
	default:
		return fmt.Errorf("cannot scan type %T into time.Time", value)
	}
}

func (s *TimeScanner) parse(v string) error {
	if v == "" || v == "0000-00-00 00:00:00" || v == "0000-00-00" {
		s.Valid = false
		return nil
	}

	layouts := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02",
		time.RFC3339Nano,
	}

	for _, layout := range layouts {
		if t, e := time.ParseInLocation(layout, v, time.Local); e == nil {
			s.Value = t
			s.Valid = true
			return nil
		}
	}
	// Fallback to UTC if Local fails or just try Parse
	for _, layout := range layouts {
		if t, e := time.Parse(layout, v); e == nil {
			s.Value = t
			s.Valid = true
			return nil
		}
	}

	return fmt.Errorf("failed to parse time: %s", v)
}

func (q *Query) scanRowWithPlan(rows *sql.Rows, dest any, plan *scanPlan) error {
	buf := scanBufferPool.Get().(*scanBuffer)
	if cap(buf.values) < len(plan.fields) {
		buf.values = make([]any, len(plan.fields))
	} else {
		buf.values = buf.values[:len(plan.fields)]
	}
	defer scanBufferPool.Put(buf)

	for i, field := range plan.fields {
		if field != nil {
			if field.Type == timeType {
				buf.values[i] = &TimeScanner{}
			} else if field.Type == timePtrType {
				buf.values[i] = &TimeScanner{}
			} else {
				buf.values[i] = reflect.New(field.Type).Interface()
			}
		} else {
			var ignore any
			buf.values[i] = &ignore
		}
	}

	if err := rows.Scan(buf.values...); err != nil {
		return fmt.Errorf("sql scan failed: %w", err)
	}

	var destValue reflect.Value
	if v, ok := dest.(reflect.Value); ok {
		destValue = v
	} else {
		destValue = reflect.ValueOf(dest)
		if destValue.Kind() == reflect.Ptr {
			destValue = destValue.Elem()
		}
	}

	for i, field := range plan.fields {
		if field != nil {
			var val reflect.Value
			if ts, ok := buf.values[i].(*TimeScanner); ok {
				if field.Type == timeType {
					if ts.Valid {
						val = reflect.ValueOf(ts.Value)
					} else {
						val = reflect.ValueOf(time.Time{})
					}
				} else { // *time.Time
					if ts.Valid {
						t := ts.Value
						val = reflect.ValueOf(&t)
					} else {
						val = reflect.Zero(field.Type)
					}
				}
			} else {
				val = reflect.ValueOf(buf.values[i]).Elem()
			}
			setFieldValue(destValue, field, val, plan, i)
		}
	}

	return nil
}

func setFieldValue(dest reflect.Value, field *model.Field, value reflect.Value, plan *scanPlan, index int) {
	f := field.Accessor(dest)
	if f.IsValid() && f.CanSet() {
		conv := plan.converters[index]
		if conv == nil {
			conv = getConverter(value.Type(), f.Type())
			plan.converters[index] = conv
		}
		conv(value, f)
	}
}

// InsertWithValidator performs an insertion after successfully validating the model.
// It returns the last inserted ID and any error encountered (including validation errors).
func (q *Query) InsertWithValidator(value any, validators ...validator.Validator) (int64, error) {
	for _, v := range validators {
		if err := v(value); err != nil {
			return 0, err
		}
	}
	return q.Insert(value)
}

// Insert inserts a new record into the database based on the provided model instance.
// It returns the last inserted ID and any error encountered.
// It also handles BeforeInsert and AfterInsert hooks, and auto-populates time fields.
func (q *Query) Insert(value any) (int64, error) {
	defer PutBuilder(q.builder)
	if q.err != nil {
		return 0, q.err
	}
	m, err := model.GetModel(value)
	if err != nil {
		return 0, fmt.Errorf("failed to get model: %w", err)
	}

	if m.HasBeforeInsert {
		if h, ok := value.(model.BeforeInserter); ok {
			if err := h.BeforeInsert(); err != nil {
				return 0, fmt.Errorf("BeforeInsert hook failed: %w", err)
			}
		}
	}

	q.builder.SetTable(m.TableName)
	cols, vals := getModelValues(m, value, false)
	sqlStr, args := q.builder.BuildInsert(cols)

	start := time.Now()
	res, err := q.executor.ExecContext(q.ctx, sqlStr, append(vals, args...)...)
	q.logSQL(sqlStr, time.Since(start), append(vals, args...)...)
	if err != nil {
		return 0, q.handleError(fmt.Errorf("Insert execution failed: %w", err))
	}

	id, _ := res.LastInsertId()

	if m.PKField != nil && m.PKField.IsAuto {
		setPKValue(value, m.PKField, id)
	}

	if m.HasAfterInsert {
		if h, ok := value.(model.AfterInserter); ok {
			if err := h.AfterInsert(id); err != nil {
				return 0, q.handleError(fmt.Errorf("AfterInsert hook failed: %w", err))
			}
		}
	}

	q.handleError(nil)
	return id, nil
}

func getModelValues(m *model.Model, value any, update bool) ([]string, []any) {
	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	var columns []string
	var args []any
	now := time.Now()

	for _, field := range m.Fields {
		if !update && field.IsAuto {
			continue
		}
		if update && field.IsPK {
			continue
		}

		fVal := field.Accessor(val)
		if !update && field.AutoTime && fVal.CanSet() {
			fVal.Set(reflect.ValueOf(now))
		} else if !update && fVal.CanSet() && field.Type.String() == "time.Time" && fVal.IsZero() {
			// Auto-fill time.Time fields that are zero on insert, if not explicitly AutoTime
			// This helps with MySQL 0000-00-00 error for non-nullable datetime columns
			// But only if it's not a pointer (pointers can be nil)
			fVal.Set(reflect.ValueOf(now))
		}
		if field.AutoUpdate && fVal.CanSet() {
			fVal.Set(reflect.ValueOf(now))
		}

		if update && fVal.IsZero() {
			continue
		}

		columns = append(columns, field.Column)
		args = append(args, fVal.Interface())
	}
	return columns, args
}

func setPKValue(value any, pkField *model.Field, id int64) {
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	f := pkField.Accessor(v)
	if f.CanSet() {
		if f.Kind() == reflect.Int64 || f.Kind() == reflect.Int {
			f.SetInt(id)
		}
	}
}

// BatchInsert inserts multiple records into the database in a single operation.
// The values parameter must be a slice of structs or pointers to structs.
// It returns the total number of rows affected and any error encountered.
// It also handles BeforeInsert and AfterInsert hooks for each record.
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
		if m.HasBeforeInsert {
			if h, ok := item.(model.BeforeInserter); ok {
				if err := h.BeforeInsert(); err != nil {
					return 0, err
				}
			}
		}

		for _, field := range m.Fields {
			if field.IsAuto {
				continue
			}
			fVal := val.Field(field.Index)
			if (field.AutoTime || field.AutoUpdate) && fVal.CanSet() {
				fVal.Set(reflect.ValueOf(now))
			} else if fVal.CanSet() && field.Type.String() == "time.Time" && fVal.IsZero() {
				// Auto-fill time.Time fields that are zero on insert for BatchInsert as well
				fVal.Set(reflect.ValueOf(now))
			}
			args = append(args, fVal.Interface())
		}
	}

	start := time.Now()
	res, err := q.executor.ExecContext(q.ctx, sqlStr, args...)
	q.logSQL(sqlStr, time.Since(start), args...)
	if err != nil {
		return 0, q.handleError(err)
	}

	totalAffected, _ := res.RowsAffected()

	// AfterInsert hooks (Batch)
	if m.HasAfterInsert {
		for i := 0; i < sliceVal.Len(); i++ {
			item := sliceVal.Index(i).Interface()
			if h, ok := item.(model.AfterInserter); ok {
				// Note: LastInsertId in batch mode is driver-dependent
				// Usually returns the first ID of the batch
				id, _ := res.LastInsertId()
				if err := h.AfterInsert(id + int64(i)); err != nil {
					return totalAffected, q.handleError(err)
				}
			}
		}
	}

	q.handleError(nil)
	return totalAffected, nil
}

// UpdateWithValidator performs an update after successfully validating the data.
// It returns the number of rows affected and any error encountered (including validation errors).
func (q *Query) UpdateWithValidator(value any, validators ...validator.Validator) (int64, error) {
	for _, v := range validators {
		if err := v(value); err != nil {
			return 0, err
		}
	}
	return q.Update(value)
}

// Update updates the records matching the query with the provided data.
// The value parameter can be a struct (updates non-zero fields) or a map[string]any.
// It returns the number of rows affected and any error encountered.
// It handles BeforeUpdate and AfterUpdate hooks for struct updates.
func (q *Query) Update(value any) (int64, error) {
	defer PutBuilder(q.builder)
	if q.err != nil {
		return 0, q.err
	}

	var data map[string]any
	var m *model.Model
	var err error

	if reflect.TypeOf(value).Kind() == reflect.Map {
		data = value.(map[string]any)
		if q.model == nil {
			return 0, fmt.Errorf("model metadata is required for map update")
		}
		m = q.model
	} else {
		m, err = model.GetModel(value)
		if err != nil {
			return 0, fmt.Errorf("failed to get model: %w", err)
		}

		if m.HasBeforeUpdate {
			if h, ok := value.(model.BeforeUpdater); ok {
				if err := h.BeforeUpdate(); err != nil {
					return 0, fmt.Errorf("BeforeUpdate hook failed: %w", err)
				}
			}
		}

		cols, vals := getModelValues(m, value, true)
		data = make(map[string]any)
		for i, col := range cols {
			data[col] = vals[i]
		}
	}

	q.builder.SetTable(m.TableName)
	sqlStr, args := q.builder.BuildUpdate(data)

	start := time.Now()
	res, err := q.executor.ExecContext(q.ctx, sqlStr, args...)
	q.logSQL(sqlStr, time.Since(start), args...)
	if err != nil {
		return 0, q.handleError(fmt.Errorf("Update execution failed: %w", err))
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return 0, q.handleError(fmt.Errorf("failed to get rows affected: %w", err))
	}

	if reflect.TypeOf(value).Kind() != reflect.Map && m != nil && m.HasAfterUpdate {
		if h, ok := value.(model.AfterUpdater); ok {
			if err := h.AfterUpdate(); err != nil {
				return 0, q.handleError(fmt.Errorf("AfterUpdate hook failed: %w", err))
			}
		}
	}

	q.handleError(nil)
	return rows, nil
}

// Delete deletes the records matching the query.
// If a model instance is provided, it uses its primary key for the deletion criteria.
// It returns the number of rows affected and any error encountered.
// It handles BeforeDelete and AfterDelete hooks if a model instance is provided.
func (q *Query) Delete(value ...any) (int64, error) {
	defer PutBuilder(q.builder)
	if q.err != nil {
		return 0, q.err
	}

	var m *model.Model
	var err error

	if len(value) > 0 {
		m, err = model.GetModel(value[0])
		if err != nil {
			return 0, fmt.Errorf("failed to get model: %w", err)
		}

		if m.HasBeforeDelete {
			if h, ok := value[0].(model.BeforeDeleter); ok {
				if err := h.BeforeDelete(); err != nil {
					return 0, fmt.Errorf("BeforeDelete hook failed: %w", err)
				}
			}
		}

		if m.PKField != nil {
			v := reflect.ValueOf(value[0])
			if v.Kind() == reflect.Ptr {
				v = v.Elem()
			}
			pkVal := v.Field(m.PKField.Index).Interface()
			q.builder.Where(q.db.dialect.Quote(m.PKField.Column)+" = ?", pkVal)
		}
	} else if q.model != nil {
		m = q.model
	} else {
		return 0, fmt.Errorf("model metadata is required for delete")
	}

	q.builder.SetTable(m.TableName)
	sqlStr, args := q.builder.BuildDelete()

	start := time.Now()
	res, err := q.executor.ExecContext(q.ctx, sqlStr, args...)
	q.logSQL(sqlStr, time.Since(start), args...)
	if err != nil {
		return 0, q.handleError(fmt.Errorf("Delete execution failed: %w", err))
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return 0, q.handleError(fmt.Errorf("failed to get rows affected: %w", err))
	}

	if len(value) > 0 && m != nil && m.HasAfterDelete {
		if h, ok := value[0].(model.AfterDeleter); ok {
			if err := h.AfterDelete(); err != nil {
				return 0, q.handleError(fmt.Errorf("AfterDelete hook failed: %w", err))
			}
		}
	}

	q.handleError(nil)
	return rows, nil
}
