package dialect

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	"github.com/shrek82/jorm/model"
)

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

func (d *mysql) Placeholder(index int) string {
	return "?"
}

func (d *mysql) GetColumnsSQL(tableName string) (string, []any) {
	return "SELECT column_name FROM information_schema.columns WHERE table_name = ?", []any{tableName}
}

func (d *mysql) AddColumnSQL(tableName string, field *model.Field) (string, []any) {
	sql := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s",
		d.Quote(tableName),
		d.Quote(field.Column),
		d.DataTypeOf(field.Type),
	)
	return sql, nil
}

func (d *mysql) ModifyColumnSQL(tableName string, field *model.Field) (string, []any) {
	sql := fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s %s",
		d.Quote(tableName),
		d.Quote(field.Column),
		d.DataTypeOf(field.Type),
	)
	return sql, nil
}

func (d *mysql) ParseColumns(rows *sql.Rows) ([]string, error) {
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

func (d *mysql) GetIndexesSQL(tableName string) (string, []any) {
	return fmt.Sprintf("SHOW INDEX FROM %s", d.Quote(tableName)), nil
}

func (d *mysql) ParseIndexes(rows *sql.Rows) (map[string][]string, error) {
	indexes := make(map[string][]string)
	for rows.Next() {
		var table, nonUnique, keyName, seqInIndex, columnName, collation, cardinality, subPart, packed, nullable, indexType, comment, indexComment, visible, expression any
		// MySQL SHOW INDEX has many columns
		err := rows.Scan(&table, &nonUnique, &keyName, &seqInIndex, &columnName, &collation, &cardinality, &subPart, &packed, &nullable, &indexType, &comment, &indexComment, &visible, &expression)
		if err != nil {
			// Older MySQL versions might have fewer columns
			err = rows.Scan(&table, &nonUnique, &keyName, &seqInIndex, &columnName, &collation, &cardinality, &subPart, &packed, &nullable, &indexType, &comment, &indexComment)
			if err != nil {
				return nil, err
			}
		}

		name := fmt.Sprintf("%v", keyName)
		col := fmt.Sprintf("%v", columnName)
		indexes[name] = append(indexes[name], col)
	}
	return indexes, nil
}

func (d *mysql) CreateIndexSQL(tableName string, indexName string, columns []string, unique bool) (string, []any) {
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
