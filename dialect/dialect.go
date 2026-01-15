package dialect

import (
	"fmt"
	"reflect"
	"strings"

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

// MySQL dialect implementation
type mysql struct{}

func (d *mysql) DataTypeOf(typ reflect.Type) string {
	switch typ.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		return "int"
	case reflect.Int64, reflect.Uint64:
		return "bigint"
	case reflect.Float32, reflect.Float64:
		return "double"
	case reflect.String:
		return "varchar(255)"
	case reflect.Struct:
		if typ.Name() == "Time" {
			return "datetime"
		}
	}
	panic(fmt.Sprintf("invalid sql type %s (%s)", typ.Name(), typ.Kind()))
}

func (d *mysql) Quote(name string) string {
	return fmt.Sprintf("`%s`", name)
}

func (d *mysql) InsertSQL(table string, columns []string) (string, []any) {
	var placeholders []string
	for range columns {
		placeholders = append(placeholders, "?")
	}
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		d.Quote(table),
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)
	return sql, nil
}

func (d *mysql) CreateTableSQL(m *model.Model) (string, []any) {
	var columns []string
	for _, field := range m.Fields {
		column := fmt.Sprintf("%s %s", d.Quote(field.Column), d.DataTypeOf(field.Type))
		if field.IsPK {
			column += " PRIMARY KEY"
		}
		if field.IsAuto {
			column += " AUTO_INCREMENT"
		}
		columns = append(columns, column)
	}
	sql := fmt.Sprintf("CREATE TABLE %s (%s)", d.Quote(m.TableName), strings.Join(columns, ", "))
	return sql, nil
}

func (d *mysql) HasTableSQL(tableName string) (string, []any) {
	return "SELECT count(*) FROM information_schema.tables WHERE table_name = ?", []any{tableName}
}

func (d *mysql) BatchInsertSQL(table string, columns []string, count int) (string, []any) {
	var rowPlaceholders []string
	for i := 0; i < count; i++ {
		var placeholders []string
		for range columns {
			placeholders = append(placeholders, "?")
		}
		rowPlaceholders = append(rowPlaceholders, "("+strings.Join(placeholders, ", ")+")")
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		d.Quote(table),
		strings.Join(columns, ", "),
		strings.Join(rowPlaceholders, ", "),
	)
	return sql, nil
}

// PostgreSQL dialect implementation
type postgres struct{}

func (d *postgres) DataTypeOf(typ reflect.Type) string {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	switch typ.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		return "integer"
	case reflect.Int64, reflect.Uint64:
		return "bigint"
	case reflect.Float32:
		return "real"
	case reflect.Float64:
		return "double precision"
	case reflect.String:
		return "varchar(255)"
	case reflect.Struct:
		if typ.Name() == "Time" {
			return "timestamp with time zone"
		}
	}
	panic(fmt.Sprintf("invalid sql type %s (%s)", typ.Name(), typ.Kind()))
}

func (d *postgres) Quote(name string) string {
	// PostgreSQL uses double quotes for identifiers
	return fmt.Sprintf(`"%s"`, name)
}

func (d *postgres) InsertSQL(table string, columns []string) (string, []any) {
	var placeholders []string
	// PostgreSQL uses $1, $2, $3... for placeholders
	for i := range columns {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
	}
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		d.Quote(table),
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)
	return sql, nil
}

func (d *postgres) CreateTableSQL(m *model.Model) (string, []any) {
	var columns []string
	for _, field := range m.Fields {
		column := fmt.Sprintf("%s %s", d.Quote(field.Column), d.DataTypeOf(field.Type))
		if field.IsPK {
			column += " PRIMARY KEY"
		}
		if field.IsAuto {
			// PostgreSQL uses SERIAL for auto-incrementing integer columns
			if strings.Contains(d.DataTypeOf(field.Type), "integer") {
				column = fmt.Sprintf("%s SERIAL", d.Quote(field.Column))
			} else {
				column += " GENERATED ALWAYS AS IDENTITY"
			}
		}
		columns = append(columns, column)
	}
	sql := fmt.Sprintf("CREATE TABLE %s (%s)", d.Quote(m.TableName), strings.Join(columns, ", "))
	return sql, nil
}

func (d *postgres) HasTableSQL(tableName string) (string, []any) {
	return "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1", []any{tableName}
}

func (d *postgres) BatchInsertSQL(table string, columns []string, count int) (string, []any) {
	var rowPlaceholders []string
	// For PostgreSQL, generate placeholders for each row
	argIndex := 1
	for i := 0; i < count; i++ {
		var placeholders []string
		for range columns {
			placeholders = append(placeholders, fmt.Sprintf("$%d", argIndex))
			argIndex++
		}
		rowPlaceholders = append(rowPlaceholders, "("+strings.Join(placeholders, ", ")+")")
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		d.Quote(table),
		strings.Join(columns, ", "),
		strings.Join(rowPlaceholders, ", "),
	)
	return sql, nil
}

// SQLite dialect implementation
type sqlite3 struct{}

func (d *sqlite3) DataTypeOf(typ reflect.Type) string {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	switch typ.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr,
		reflect.Int64, reflect.Uint64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "real"
	case reflect.String:
		return "text"
	case reflect.Struct:
		if typ.Name() == "Time" {
			return "datetime"
		}
	}
	panic(fmt.Sprintf("invalid sql type %s (%s)", typ.Name(), typ.Kind()))
}

func (d *sqlite3) Quote(name string) string {
	return fmt.Sprintf("`%s`", name)
}

func (d *sqlite3) InsertSQL(table string, columns []string) (string, []any) {
	var placeholders []string
	for range columns {
		placeholders = append(placeholders, "?")
	}
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		d.Quote(table),
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)
	return sql, nil
}

func (d *sqlite3) CreateTableSQL(m *model.Model) (string, []any) {
	var columns []string
	for _, field := range m.Fields {
		column := fmt.Sprintf("%s %s", d.Quote(field.Column), d.DataTypeOf(field.Type))
		if field.IsPK {
			column += " PRIMARY KEY"
		}
		if field.IsAuto {
			column += " AUTOINCREMENT"
		}
		columns = append(columns, column)
	}
	sql := fmt.Sprintf("CREATE TABLE %s (%s)", d.Quote(m.TableName), strings.Join(columns, ", "))
	return sql, nil
}

func (d *sqlite3) HasTableSQL(tableName string) (string, []any) {
	return "SELECT count(*) FROM sqlite_master WHERE type='table' AND name = ?", []any{tableName}
}

func (d *sqlite3) BatchInsertSQL(table string, columns []string, count int) (string, []any) {
	var rowPlaceholders []string
	for i := 0; i < count; i++ {
		var placeholders []string
		for range columns {
			placeholders = append(placeholders, "?")
		}
		rowPlaceholders = append(rowPlaceholders, "("+strings.Join(placeholders, ", ")+")")
	}

	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES %s",
		d.Quote(table),
		strings.Join(columns, ", "),
		strings.Join(rowPlaceholders, ", "),
	)
	return sql, nil
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