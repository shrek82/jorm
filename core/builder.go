package core

import (
	"strings"
	"sync"

	"github.com/shrek82/jorm/dialect"
	"github.com/shrek82/jorm/query"
)

// Builder defines the interface for building SQL statements.
// It abstracts the SQL generation process from the execution logic.
type Builder interface {
	SetTable(name string) Builder
	Select(columns ...string) Builder
	Where(cond string, args ...any) Builder
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
	dialect dialect.Dialect
	table   string
	clauses map[query.ClauseType]*query.Clause
}

var builderPool = sync.Pool{
	New: func() any {
		return &sqlBuilder{
			clauses: make(map[query.ClauseType]*query.Clause),
		}
	},
}

// NewBuilder creates a new sqlBuilder instance with the given dialect.
func NewBuilder(d dialect.Dialect) Builder {
	b := builderPool.Get().(*sqlBuilder)
	b.dialect = d
	// Reset builder
	b.table = ""
	for k := range b.clauses {
		delete(b.clauses, k)
	}
	return b
}

// SetTable sets the table name for the current SQL statement.
func (b *sqlBuilder) SetTable(name string) Builder {
	b.table = name
	return b
}

// Select adds the SELECT clause with specified columns.
func (b *sqlBuilder) Select(columns ...string) Builder {
	if c, ok := b.clauses[query.SELECT]; ok {
		existing := c.Value[0].([]string)
		existing = append(existing, columns...)
		c.Value[0] = existing
	} else {
		b.clauses[query.SELECT] = &query.Clause{Type: query.SELECT, Value: []any{columns}}
	}
	return b
}

// Where adds the WHERE clause with condition and arguments.
func (b *sqlBuilder) Where(cond string, args ...any) Builder {
	if c, ok := b.clauses[query.WHERE]; ok {
		conds := c.Value[0].([]string)
		conds = append(conds, cond)
		c.Value[0] = conds

		existingArgs := c.Value[1].([]any)
		existingArgs = append(existingArgs, args...)
		c.Value[1] = existingArgs
	} else {
		b.clauses[query.WHERE] = &query.Clause{
			Type: query.WHERE,
			Value: []any{
				[]string{cond},
				append([]any{}, args...),
			},
		}
	}
	return b
}

func (b *sqlBuilder) Join(table, joinType, on string) Builder {
	jt := strings.TrimSpace(joinType)
	if jt == "" {
		jt = "INNER"
	}
	quotedTable := b.dialect.Quote(table)
	clause := jt + " JOIN " + quotedTable + " ON " + on
	if c, ok := b.clauses[query.JOIN]; ok {
		joins := c.Value[0].([]string)
		joins = append(joins, clause)
		c.Value[0] = joins
	} else {
		b.clauses[query.JOIN] = &query.Clause{Type: query.JOIN, Value: []any{[]string{clause}}}
	}
	return b
}

// OrderBy adds the ORDER BY clause.
func (b *sqlBuilder) OrderBy(columns ...string) Builder {
	b.clauses[query.ORDERBY] = &query.Clause{Type: query.ORDERBY, Value: []any{columns}}
	return b
}

// Limit adds the LIMIT clause.
func (b *sqlBuilder) Limit(n int) Builder {
	b.clauses[query.LIMIT] = &query.Clause{Type: query.LIMIT, Value: []any{n}}
	return b
}

// Offset adds the OFFSET clause.
func (b *sqlBuilder) Offset(n int) Builder {
	b.clauses[query.OFFSET] = &query.Clause{Type: query.OFFSET, Value: []any{n}}
	return b
}

// BuildSelect generates the complete SELECT SQL statement and its arguments.
func (b *sqlBuilder) BuildSelect() (string, []any) {
	var sqls []string
	var args []any

	// SELECT
	if c, ok := b.clauses[query.SELECT]; ok {
		s, a := c.Build()
		sqls = append(sqls, s)
		args = append(args, a...)
	} else {
		sqls = append(sqls, "SELECT *")
	}

	// FROM
	sqls = append(sqls, "FROM "+b.dialect.Quote(b.table))

	if c, ok := b.clauses[query.JOIN]; ok {
		s, a := c.Build()
		sqls = append(sqls, s)
		args = append(args, a...)
	}

	// WHERE, ORDER BY, LIMIT, OFFSET
	types := []query.ClauseType{query.WHERE, query.ORDERBY, query.LIMIT, query.OFFSET}
	for _, t := range types {
		if c, ok := b.clauses[t]; ok {
			s, a := c.Build()
			sqls = append(sqls, s)
			args = append(args, a...)
		}
	}

	return strings.Join(sqls, " "), args
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

	var sets []string
	for col, val := range data {
		sets = append(sets, b.dialect.Quote(col)+" = ?")
		args = append(args, val)
	}
	sqls = append(sqls, strings.Join(sets, ", "))

	if c, ok := b.clauses[query.WHERE]; ok {
		s, a := c.Build()
		sqls = append(sqls, s)
		args = append(args, a...)
	}

	return strings.Join(sqls, " "), args
}

// BuildDelete generates the DELETE SQL statement.
func (b *sqlBuilder) BuildDelete() (string, []any) {
	var sqls []string
	var args []any

	sqls = append(sqls, "DELETE FROM", b.dialect.Quote(b.table))

	if c, ok := b.clauses[query.WHERE]; ok {
		s, a := c.Build()
		sqls = append(sqls, s)
		args = append(args, a...)
	}

	return strings.Join(sqls, " "), args
}
