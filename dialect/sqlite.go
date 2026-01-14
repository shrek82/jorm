package dialect

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/shrek82/jorm/model"
)

type sqlite3 struct{}

func init() {
	Register("sqlite3", &sqlite3{})
}

func (d *sqlite3) DataTypeOf(typ reflect.Type) string {
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
