package dialect

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/shrek82/jorm/model"
)

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

func (d *postgres) Placeholder(index int) string {
	return fmt.Sprintf("$%d", index)
}

func (d *postgres) GetColumnsSQL(tableName string) (string, []any) {
	return "SELECT column_name FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1", []any{tableName}
}

func (d *postgres) AddColumnSQL(tableName string, field *model.Field) (string, []any) {
	sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
		d.Quote(tableName),
		d.Quote(field.Column),
		d.DataTypeOf(field.Type),
	)
	return sql, nil
}

func (d *postgres) ModifyColumnSQL(tableName string, field *model.Field) (string, []any) {
	sql := fmt.Sprintf("ALTER TABLE %s ALTER COLUMN %s TYPE %s",
		d.Quote(tableName),
		d.Quote(field.Column),
		d.DataTypeOf(field.Type),
	)
	return sql, nil
}

func (d *postgres) ParseColumns(rows *sql.Rows) ([]string, error) {
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

func (d *postgres) GetIndexesSQL(tableName string) (string, []any) {
	return "SELECT indexname, indexdef FROM pg_indexes WHERE schemaname = 'public' AND tablename = $1", []any{tableName}
}

func (d *postgres) ParseIndexes(rows *sql.Rows) (map[string][]string, error) {
	indexes := make(map[string][]string)
	for rows.Next() {
		var name, def string
		if err := rows.Scan(&name, &def); err != nil {
			return nil, err
		}
		// Extract columns from index definition
		// e.g. "CREATE UNIQUE INDEX idx_name ON table_name USING btree (col1, col2)"
		start := strings.Index(def, "(")
		end := strings.LastIndex(def, ")")
		if start != -1 && end != -1 && end > start {
			colsPart := def[start+1 : end]
			cols := strings.Split(colsPart, ",")
			for _, col := range cols {
				indexes[name] = append(indexes[name], strings.TrimSpace(col))
			}
		}
	}
	return indexes, nil
}

func (d *postgres) CreateIndexSQL(tableName string, indexName string, columns []string, unique bool) (string, []any) {
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
