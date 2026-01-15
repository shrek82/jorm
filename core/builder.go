package core

import (
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/shrek82/jorm/dialect"
)

// Builder defines the interface for building SQL statements.
// It abstracts the SQL generation process from the execution logic.
type Builder interface {
	SetTable(name string) Builder
	Alias(alias string) Builder
	Select(columns ...string) Builder
	Where(cond string, args ...any) Builder
	OrWhere(cond string, args ...any) Builder
	WhereIn(column string, values any) Builder
	Join(table, joinType, on string) Builder
	OrderBy(columns ...string) Builder
	Limit(n int) Builder
	Offset(n int) Builder
	BuildSelect() (string, []any)
	BuildInsert(columns []string) (string, []any)
	BuildUpdate(data map[string]any) (string, []any)
	BuildDelete() (string, []any)
}

// sqlBuilder is the default implementation of the Builder interface.
type sqlBuilder struct {
	dialect    dialect.Dialect
	table      string
	alias      string
	selectCols []string
	whereExpr  string
	whereArgs  []any
	joins      []string
	orderBy    []string
	limitSet   bool
	limit      int
	offsetSet  bool
	offset     int
}

var builderPool = sync.Pool{
	New: func() any {
		return &sqlBuilder{}
	},
}

// NewBuilder creates a new sqlBuilder instance with the given dialect.
func NewBuilder(d dialect.Dialect) Builder {
	b := builderPool.Get().(*sqlBuilder)
	b.dialect = d
	b.table = ""
	b.alias = ""
	b.selectCols = b.selectCols[:0]
	b.whereExpr = ""
	b.whereArgs = b.whereArgs[:0]
	b.joins = b.joins[:0]
	b.orderBy = b.orderBy[:0]
	b.limitSet = false
	b.limit = 0
	b.offsetSet = false
	b.offset = 0
	return b
}

// SetTable sets the table name for the current SQL statement.
func (b *sqlBuilder) SetTable(name string) Builder {
	b.table = name
	return b
}

func (b *sqlBuilder) Alias(alias string) Builder {
	b.alias = strings.TrimSpace(alias)
	return b
}

// Select adds the SELECT clause with specified columns.
func (b *sqlBuilder) Select(columns ...string) Builder {
	b.selectCols = append(b.selectCols, columns...)
	return b
}

// Where adds the WHERE clause with condition and arguments.
func (b *sqlBuilder) Where(cond string, args ...any) Builder {
	if cond == "" {
		return b
	}
	if b.whereExpr == "" {
		b.whereExpr = "(" + cond + ")"
	} else {
		b.whereExpr = b.whereExpr + " AND (" + cond + ")"
	}
	b.whereArgs = append(b.whereArgs, args...)
	return b
}

func (b *sqlBuilder) OrWhere(cond string, args ...any) Builder {
	if cond == "" {
		return b
	}
	if b.whereExpr == "" {
		b.whereExpr = "(" + cond + ")"
	} else {
		b.whereExpr = b.whereExpr + " OR (" + cond + ")"
	}
	b.whereArgs = append(b.whereArgs, args...)
	return b
}

func (b *sqlBuilder) WhereIn(column string, values any) Builder {
	v := reflect.ValueOf(values)
	if !v.IsValid() {
		return b
	}
	kind := v.Kind()
	if kind != reflect.Slice && kind != reflect.Array {
		return b.Where(column+" IN (?)", values)
	}
	if v.Len() == 0 {
		return b.Where("1 = 0")
	}

	// For WhereIn, we don't know the final argument index yet because
	// BuildSelect/BuildUpdate will handle the final placeholder generation.
	// However, the current implementation of Where() and Build* methods
	// expects placeholders to be in the condition string.
	// This is a bit tricky with positional placeholders ($1, $2).
	// A better way is to use a special placeholder in Where strings
	// and replace them during Build.

	placeholders := make([]string, v.Len())
	args := make([]any, 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		placeholders[i] = "?"
		args = append(args, v.Index(i).Interface())
	}
	cond := column + " IN (" + strings.Join(placeholders, ", ") + ")"
	return b.Where(cond, args...)
}

func (b *sqlBuilder) Join(table, joinType, on string) Builder {
	jt := strings.TrimSpace(joinType)
	if jt == "" {
		jt = "INNER"
	}
	quotedTable := b.dialect.Quote(table)
	clause := jt + " JOIN " + quotedTable + " ON " + on
	b.joins = append(b.joins, clause)
	return b
}

// OrderBy adds the ORDER BY clause.
func (b *sqlBuilder) OrderBy(columns ...string) Builder {
	b.orderBy = append(b.orderBy, columns...)
	return b
}

// Limit adds the LIMIT clause.
func (b *sqlBuilder) Limit(n int) Builder {
	b.limitSet = true
	b.limit = n
	return b
}

// Offset adds the OFFSET clause.
func (b *sqlBuilder) Offset(n int) Builder {
	b.offsetSet = true
	b.offset = n
	return b
}

func (b *sqlBuilder) replacePlaceholders(sql string) string {
	if strings.Contains(sql, "?") {
		index := 1
		for {
			newSQL := strings.Replace(sql, "?", b.dialect.Placeholder(index), 1)
			if newSQL == sql {
				break
			}
			sql = newSQL
			index++
		}
	}
	return sql
}

// BuildSelect generates the complete SELECT SQL statement and its arguments.
func (b *sqlBuilder) BuildSelect() (string, []any) {
	var sqls []string
	var args []any

	// SELECT
	if len(b.selectCols) > 0 {
		sqls = append(sqls, "SELECT "+strings.Join(b.selectCols, ", "))
	} else {
		sqls = append(sqls, "SELECT *")
	}

	// FROM
	from := "FROM " + b.dialect.Quote(b.table)
	if b.alias != "" {
		from += " " + b.alias
	}
	sqls = append(sqls, from)

	if len(b.joins) > 0 {
		sqls = append(sqls, strings.Join(b.joins, " "))
	}

	if b.whereExpr != "" {
		sqls = append(sqls, "WHERE "+b.whereExpr)
		args = append(args, b.whereArgs...)
	}

	if len(b.orderBy) > 0 {
		sqls = append(sqls, "ORDER BY "+strings.Join(b.orderBy, ", "))
	}

	if b.limitSet {
		sqls = append(sqls, "LIMIT ?")
		args = append(args, b.limit)
	}

	if b.offsetSet {
		sqls = append(sqls, "OFFSET ?")
		args = append(args, b.offset)
	}

	sql := strings.Join(sqls, " ")
	return b.replacePlaceholders(sql), args
}

// PutBuilder returns a sqlBuilder to the pool for reuse.
func PutBuilder(b Builder) {
	if sb, ok := b.(*sqlBuilder); ok {
		builderPool.Put(sb)
	}
}

// BuildInsert generates the INSERT SQL statement.
func (b *sqlBuilder) BuildInsert(columns []string) (string, []any) {
	return b.dialect.InsertSQL(b.table, columns)
}

// BuildUpdate generates the UPDATE SQL statement.
func (b *sqlBuilder) BuildUpdate(data map[string]any) (string, []any) {
	var sqls []string
	var args []any

	sqls = append(sqls, "UPDATE", b.dialect.Quote(b.table), "SET")

	// Sort columns to ensure deterministic SQL generation
	columns := make([]string, 0, len(data))
	for col := range data {
		columns = append(columns, col)
	}
	sort.Strings(columns)

	var sets []string
	for _, col := range columns {
		sets = append(sets, b.dialect.Quote(col)+" = ?")
		args = append(args, data[col])
	}
	sqls = append(sqls, strings.Join(sets, ", "))

	if b.whereExpr != "" {
		sqls = append(sqls, "WHERE "+b.whereExpr)
		args = append(args, b.whereArgs...)
	}

	sql := strings.Join(sqls, " ")
	return b.replacePlaceholders(sql), args
}

// BuildDelete generates the DELETE SQL statement.
func (b *sqlBuilder) BuildDelete() (string, []any) {
	var sqls []string
	var args []any

	sqls = append(sqls, "DELETE FROM", b.dialect.Quote(b.table))

	if b.whereExpr != "" {
		sqls = append(sqls, "WHERE "+b.whereExpr)
		args = append(args, b.whereArgs...)
	}

	sql := strings.Join(sqls, " ")
	return b.replacePlaceholders(sql), args
}
