package core

import (
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/shrek82/jorm/dialect"
)

// Builder defines the interface for building SQL statements.
// It provides a fluent API for constructing complex queries and handles
// dialect-specific syntax like quoting and placeholders.
type Builder interface {
	// SetTable sets the target table for the SQL statement.
	SetTable(name string) Builder
	// Alias sets a table alias (e.g., "users AS u").
	Alias(alias string) Builder
	// Select specifies columns to retrieve (e.g., "id", "name").
	Select(columns ...string) Builder
	// Where adds an AND condition to the WHERE clause.
	Where(cond string, args ...any) Builder
	// OrWhere adds an OR condition to the WHERE clause.
	OrWhere(cond string, args ...any) Builder
	// WhereIn adds an IN condition for a column and a slice of values.
	WhereIn(column string, values any) Builder
	// Joins adds a raw JOIN clause (e.g., "JOIN orders ON orders.user_id = users.id").
	Joins(query string, args ...any) Builder
	// GroupBy adds columns for the GROUP BY clause.
	GroupBy(columns ...string) Builder
	// Having adds an AND condition to the HAVING clause.
	Having(cond string, args ...any) Builder
	// OrderBy adds columns for the ORDER BY clause (e.g., "id DESC").
	OrderBy(columns ...string) Builder
	// Limit sets the maximum number of rows to return.
	Limit(n int) Builder
	// Offset sets the number of rows to skip.
	Offset(n int) Builder
	// BuildSelect generates the final SELECT statement and its arguments.
	BuildSelect() (string, []any)
	// BuildInsert generates the final INSERT statement and its arguments.
	BuildInsert(columns []string) (string, []any)
	// BuildUpdate generates the final UPDATE statement and its arguments.
	BuildUpdate(data map[string]any) (string, []any)
	// BuildDelete generates the final DELETE statement and its arguments.
	BuildDelete() (string, []any)
	// Clone creates a deep copy of the builder.
	Clone() Builder
}

// sqlBuilder is the default implementation of the Builder interface.
// It tracks query components and assembles them into a SQL string.
type sqlBuilder struct {
	dialect    dialect.Dialect // Database-specific dialect
	table      string          // Target table name
	alias      string          // Table alias
	selectCols []string        // Columns to select
	whereExpr  string          // WHERE clause expression
	whereArgs  []any           // WHERE clause arguments
	joins      []string        // JOIN clauses
	joinArgs   []any           // JOIN clause arguments
	groupBy    []string        // GROUP BY columns
	havingExpr string          // HAVING clause expression
	havingArgs []any           // HAVING clause arguments
	orderBy    []string        // ORDER BY columns
	limitSet   bool            // Whether limit is set
	limit      int             // LIMIT value
	offsetSet  bool            // Whether offset is set
	offset     int             // OFFSET value
	sb         strings.Builder // Reusable string builder
}

var builderPool = sync.Pool{
	New: func() any {
		return &sqlBuilder{}
	},
}

// NewBuilder creates a new sqlBuilder instance with the given dialect.
func NewBuilder(d dialect.Dialect) Builder {
	b := builderPool.Get().(*sqlBuilder)
	b.Reset(d)
	return b
}

// Reset clears all builder state and prepares it for a new query with the given dialect.
func (b *sqlBuilder) Reset(d dialect.Dialect) {
	b.dialect = d
	b.table = ""
	b.alias = ""
	b.selectCols = b.selectCols[:0]
	b.whereExpr = ""
	b.whereArgs = b.whereArgs[:0]
	b.joins = b.joins[:0]
	b.joinArgs = b.joinArgs[:0]
	b.groupBy = b.groupBy[:0]
	b.havingExpr = ""
	b.havingArgs = b.havingArgs[:0]
	b.orderBy = b.orderBy[:0]
	b.limitSet = false
	b.limit = 0
	b.offsetSet = false
	b.offset = 0
	b.sb.Reset()
}

// Clone creates a deep copy of the builder.
func (b *sqlBuilder) Clone() Builder {
	nb := builderPool.Get().(*sqlBuilder)
	nb.Reset(b.dialect)

	nb.table = b.table
	nb.alias = b.alias

	if len(b.selectCols) > 0 {
		nb.selectCols = append(nb.selectCols, b.selectCols...)
	}

	nb.whereExpr = b.whereExpr
	if len(b.whereArgs) > 0 {
		nb.whereArgs = append(nb.whereArgs, b.whereArgs...)
	}

	if len(b.joins) > 0 {
		nb.joins = append(nb.joins, b.joins...)
	}
	if len(b.joinArgs) > 0 {
		nb.joinArgs = append(nb.joinArgs, b.joinArgs...)
	}

	if len(b.groupBy) > 0 {
		nb.groupBy = append(nb.groupBy, b.groupBy...)
	}

	nb.havingExpr = b.havingExpr
	if len(b.havingArgs) > 0 {
		nb.havingArgs = append(nb.havingArgs, b.havingArgs...)
	}

	if len(b.orderBy) > 0 {
		nb.orderBy = append(nb.orderBy, b.orderBy...)
	}

	nb.limitSet = b.limitSet
	nb.limit = b.limit
	nb.offsetSet = b.offsetSet
	nb.offset = b.offset

	return nb
}

// SetTable sets the table name. for the current SQL statement.
func (b *sqlBuilder) SetTable(name string) Builder {
	b.table = name
	return b
}

// Alias sets a table alias for the query.
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

// OrWhere adds an OR condition to the WHERE clause.
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

// WhereIn adds an IN condition for the specified column and values.
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

// Joins adds a raw JOIN clause to the query.
func (b *sqlBuilder) Joins(query string, args ...any) Builder {
	if !isValidJoinClause(query) {
		panic("invalid join clause: " + query)
	}
	b.joins = append(b.joins, query)
	b.joinArgs = append(b.joinArgs, args...)
	return b
}

func (b *sqlBuilder) GroupBy(columns ...string) Builder {
	b.groupBy = append(b.groupBy, columns...)
	return b
}

// Having adds a condition to the HAVING clause.
func (b *sqlBuilder) Having(cond string, args ...any) Builder {
	if cond == "" {
		return b
	}
	if b.havingExpr == "" {
		b.havingExpr = "(" + cond + ")"
	} else {
		b.havingExpr = b.havingExpr + " AND (" + cond + ")"
	}
	b.havingArgs = append(b.havingArgs, args...)
	return b
}

func isValidJoinClause(query string) bool {
	upper := strings.ToUpper(query)
	// Check for forbidden characters/sequences that indicate multiple statements or comments
	forbidden := []string{";", "--", "/*", "*/"}
	for _, s := range forbidden {
		if strings.Contains(upper, s) {
			return false
		}
	}

	// Check for dangerous SQL keywords
	keywords := []string{"DROP ", "DELETE ", "UPDATE ", "INSERT ", "TRUNCATE ", "ALTER "}
	for _, k := range keywords {
		if strings.Contains(upper, k) {
			return false
		}
	}

	// A basic JOIN clause should contain "JOIN"
	return strings.Contains(upper, "JOIN")
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
	if !strings.Contains(sql, "?") {
		return sql
	}

	// Reuse b.sb for building the final string with correct placeholders.
	// Since sql is the result of b.sb.String() from Build* methods, it is safe to reset b.sb.
	b.sb.Reset()

	index := 1
	for {
		idx := strings.Index(sql, "?")
		if idx == -1 {
			b.sb.WriteString(sql)
			break
		}

		b.sb.WriteString(sql[:idx])
		b.sb.WriteString(b.dialect.Placeholder(index))
		sql = sql[idx+1:]
		index++
	}
	return b.sb.String()
}

// BuildSelect generates the complete SELECT SQL statement and its arguments.
func (b *sqlBuilder) BuildSelect() (string, []any) {
	b.sb.Reset()

	argCount := len(b.joinArgs) + len(b.whereArgs) + len(b.havingArgs)
	if b.limitSet {
		argCount++
	}
	if b.offsetSet {
		argCount++
	}
	args := make([]any, 0, argCount)

	// SELECT
	b.sb.WriteString("SELECT ")
	if len(b.selectCols) > 0 {
		for i, col := range b.selectCols {
			if i > 0 {
				b.sb.WriteString(", ")
			}
			b.sb.WriteString(col)
		}
	} else {
		b.sb.WriteString("*")
	}

	// FROM
	b.sb.WriteString(" FROM ")
	b.sb.WriteString(b.dialect.Quote(b.table))
	if b.alias != "" {
		b.sb.WriteString(" ")
		b.sb.WriteString(b.alias)
	}

	if len(b.joins) > 0 {
		b.sb.WriteString(" ")
		b.sb.WriteString(strings.Join(b.joins, " "))
		args = append(args, b.joinArgs...)
	}

	if b.whereExpr != "" {
		b.sb.WriteString(" WHERE ")
		b.sb.WriteString(b.whereExpr)
		args = append(args, b.whereArgs...)
	}

	if len(b.groupBy) > 0 {
		b.sb.WriteString(" GROUP BY ")
		b.sb.WriteString(strings.Join(b.groupBy, ", "))
	}

	if b.havingExpr != "" {
		b.sb.WriteString(" HAVING ")
		b.sb.WriteString(b.havingExpr)
		args = append(args, b.havingArgs...)
	}

	if len(b.orderBy) > 0 {
		b.sb.WriteString(" ORDER BY ")
		b.sb.WriteString(strings.Join(b.orderBy, ", "))
	}

	if b.limitSet {
		b.sb.WriteString(" LIMIT ?")
		args = append(args, b.limit)
	}

	if b.offsetSet {
		b.sb.WriteString(" OFFSET ?")
		args = append(args, b.offset)
	}

	return b.replacePlaceholders(b.sb.String()), args
}

// PutBuilder returns a sqlBuilder to the pool for reuse.
func PutBuilder(b Builder) {
	if sb, ok := b.(*sqlBuilder); ok {
		sb.Reset(nil)
		builderPool.Put(sb)
	}
}

// BuildInsert generates the INSERT SQL statement.
func (b *sqlBuilder) BuildInsert(columns []string) (string, []any) {
	return b.dialect.InsertSQL(b.table, columns)
}

// BuildUpdate generates the UPDATE SQL statement.
func (b *sqlBuilder) BuildUpdate(data map[string]any) (string, []any) {
	b.sb.Reset()

	args := make([]any, 0, len(data)+len(b.whereArgs))

	b.sb.WriteString("UPDATE ")
	b.sb.WriteString(b.dialect.Quote(b.table))
	b.sb.WriteString(" SET ")

	// Sort columns to ensure deterministic SQL generation
	columns := make([]string, 0, len(data))
	for col := range data {
		columns = append(columns, col)
	}
	sort.Strings(columns)

	for i, col := range columns {
		if i > 0 {
			b.sb.WriteString(", ")
		}
		b.sb.WriteString(b.dialect.Quote(col))
		b.sb.WriteString(" = ?")
		args = append(args, data[col])
	}

	if b.whereExpr != "" {
		b.sb.WriteString(" WHERE ")
		b.sb.WriteString(b.whereExpr)
		args = append(args, b.whereArgs...)
	}

	return b.replacePlaceholders(b.sb.String()), args
}

// BuildDelete generates the DELETE SQL statement.
func (b *sqlBuilder) BuildDelete() (string, []any) {
	b.sb.Reset()
	args := make([]any, 0, len(b.whereArgs))

	b.sb.WriteString("DELETE FROM ")
	b.sb.WriteString(b.dialect.Quote(b.table))

	if b.whereExpr != "" {
		b.sb.WriteString(" WHERE ")
		b.sb.WriteString(b.whereExpr)
		args = append(args, b.whereArgs...)
	}

	return b.replacePlaceholders(b.sb.String()), args
}
