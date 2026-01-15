package dialect

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/shrek82/jorm/model"
)

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

func (d *sqlite3) Placeholder(index int) string {
	return "?"
}

func (d *sqlite3) GetColumnsSQL(tableName string) (string, []any) {
	return fmt.Sprintf("PRAGMA table_info(%s)", d.Quote(tableName)), nil
}

func (d *sqlite3) AddColumnSQL(tableName string, field *model.Field) (string, []any) {
	sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
		d.Quote(tableName),
		d.Quote(field.Column),
		d.DataTypeOf(field.Type),
	)
	return sql, nil
}

func (d *sqlite3) ModifyColumnSQL(tableName string, field *model.Field) (string, []any) {
	// SQLite does not support MODIFY COLUMN directly.
	// This usually requires creating a new table and copying data.
	// For now, we return a no-op or error-prone SQL.
	return "", nil
}

func (d *sqlite3) ParseColumns(rows *sql.Rows) ([]string, error) {
	var columns []string
	for rows.Next() {
		var cid int
		var name string
		var typ string
		var notnull int
		var dfltValue any
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}
	return columns, nil
}

func (d *sqlite3) GetIndexesSQL(tableName string) (string, []any) {
	return fmt.Sprintf("PRAGMA index_list(%s)", d.Quote(tableName)), nil
}

func (d *sqlite3) ParseIndexes(rows *sql.Rows) (map[string][]string, error) {
	indexes := make(map[string][]string)
	var indexNames []string

	for rows.Next() {
		var seq int
		var name string
		var unique int
		var origin string
		var partial int
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return nil, err
		}
		indexNames = append(indexNames, name)
	}

	// For each index, get its columns
	// Note: This is a bit inefficient as we need to query for each index,
	// but PRAGMA doesn't provide a way to get all at once easily.
	// In JORM's current architecture, we'll do this for simplicity.
	// A better way would be to have access to the db pool here, but we don't.
	// So we'll return the names for now and handle column fetching in syncIndexes if needed,
	// OR we assume index name matches our convention.
	// For SQLite, we can just return the index names as keys.
	for _, name := range indexNames {
		indexes[name] = []string{} // Columns will be empty for now
	}

	return indexes, nil
}

func (d *sqlite3) CreateIndexSQL(tableName string, indexName string, columns []string, unique bool) (string, []any) {
	uniqueStr := ""
	if unique {
		uniqueStr = "UNIQUE "
	}
	sql := fmt.Sprintf("CREATE %sINDEX %s ON %s (%s)",
		uniqueStr,
		d.Quote(indexName),
		d.Quote(tableName),
		strings.Join(columns, ", "),
	)
	return sql, nil
}
