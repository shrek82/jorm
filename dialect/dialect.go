package dialect

import (
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
