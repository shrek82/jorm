package dialect

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/shrek82/jorm/model"
)

type sqlserver struct{}

func (d *sqlserver) DataTypeOf(typ reflect.Type) string {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	switch typ.Kind() {
	case reflect.Bool:
		return "bit"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		return "int"
	case reflect.Int64, reflect.Uint64:
		return "bigint"
	case reflect.Float32:
		return "real"
	case reflect.Float64:
		return "float"
	case reflect.String:
		return "nvarchar(255)"
	case reflect.Struct:
		if typ.Name() == "Time" {
			return "datetime2"
		}
	}
	panic(fmt.Sprintf("invalid sql type %s (%s)", typ.Name(), typ.Kind()))
}

func (d *sqlserver) Quote(name string) string {
	return fmt.Sprintf("[%s]", name)
}

func (d *sqlserver) InsertSQL(table string, columns []string) (string, []any) {
	var placeholders []string
	for i := range columns {
		placeholders = append(placeholders, fmt.Sprintf("@p%d", i+1))
	}
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		d.Quote(table),
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)
	return sql, nil
}

func (d *sqlserver) CreateTableSQL(m *model.Model) (string, []any) {
	var columns []string
	for _, field := range m.Fields {
		column := fmt.Sprintf("%s %s", d.Quote(field.Column), d.DataTypeOf(field.Type))
		if field.IsPK {
			column += " PRIMARY KEY"
		}
		if field.IsAuto {
			column += " IDENTITY(1,1)"
		}
		columns = append(columns, column)
	}
	sql := fmt.Sprintf("CREATE TABLE %s (%s)", d.Quote(m.TableName), strings.Join(columns, ", "))
	return sql, nil
}

func (d *sqlserver) HasTableSQL(tableName string) (string, []any) {
	return "SELECT count(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME = @p1", []any{tableName}
}

func (d *sqlserver) BatchInsertSQL(table string, columns []string, count int) (string, []any) {
	var rowPlaceholders []string
	argIndex := 1
	for i := 0; i < count; i++ {
		var placeholders []string
		for range columns {
			placeholders = append(placeholders, fmt.Sprintf("@p%d", argIndex))
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

func (d *sqlserver) Placeholder(index int) string {
	return fmt.Sprintf("@p%d", index)
}

func (d *sqlserver) GetColumnsSQL(tableName string) (string, []any) {
	return "SELECT column_name FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = @p1", []any{tableName}
}

func (d *sqlserver) AddColumnSQL(tableName string, field *model.Field) (string, []any) {
	sql := fmt.Sprintf("ALTER TABLE %s ADD %s %s",
		d.Quote(tableName),
		d.Quote(field.Column),
		d.DataTypeOf(field.Type),
	)
	return sql, nil
}

func (d *sqlserver) ModifyColumnSQL(tableName string, field *model.Field) (string, []any) {
	sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s %s",
		d.Quote(tableName),
		d.Quote(field.Column),
		d.DataTypeOf(field.Type),
	)
	return sql, nil
}

func (d *sqlserver) ParseColumns(rows *sql.Rows) ([]string, error) {
	var columns []string
	for rows.Next() {
		var colName string
		if err := rows.Scan(&colName); err != nil {
			return nil, err
		}
		columns = append(columns, colName)
	}
	return columns, nil
}
