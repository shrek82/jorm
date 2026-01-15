package dialect

import (
	"database/sql"
	"reflect"

	"github.com/shrek82/jorm/model"
)

// Dialect represents the interface for database-specific SQL generation and type mapping.
// Each database (MySQL, SQLite, etc.) must implement this interface to be supported.
type Dialect interface {
	// DataTypeOf returns the database-specific data type for a Go reflect.Type
	DataTypeOf(typ reflect.Type) string
	// Quote wraps a name (table or column) in database-specific quotes
	Quote(name string) string
	// InsertSQL generates the INSERT statement for the given table and columns
	InsertSQL(table string, columns []string) (string, []any)
	// CreateTableSQL generates the CREATE TABLE statement for the given model
	CreateTableSQL(m *model.Model) (string, []any)
	// HasTableSQL generates the SQL to check if a table exists
	HasTableSQL(tableName string) (string, []any)
	// BatchInsertSQL generates a single SQL statement for multiple rows
	BatchInsertSQL(table string, columns []string, count int) (string, []any)
	// Placeholder returns the database-specific placeholder for a given index (1-based)
	Placeholder(index int) string
	// GetColumnsSQL generates the SQL to get columns of a table
	GetColumnsSQL(tableName string) (string, []any)
	// AddColumnSQL generates the SQL to add a column to a table
	AddColumnSQL(tableName string, field *model.Field) (string, []any)
	// ModifyColumnSQL generates the SQL to modify a column in a table
	ModifyColumnSQL(tableName string, field *model.Field) (string, []any)
	// ParseColumns parses the rows from GetColumnsSQL into a slice of column names
	ParseColumns(rows *sql.Rows) ([]string, error)
}

var dialects = make(map[string]Dialect)

// Register registers a new dialect for a given driver name
func Register(name string, d Dialect) {
	dialects[name] = d
}

// Get retrieves a registered dialect by driver name
func Get(name string) (Dialect, bool) {
	d, ok := dialects[name]
	return d, ok
}

// Register built-in dialects
func init() {
	// Register MySQL dialect
	Register("mysql", &mysql{})

	// Register PostgreSQL dialect
	Register("postgres", &postgres{})

	// Register SQLite dialect
	Register("sqlite3", &sqlite3{})
}
