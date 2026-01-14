package model

import (
	"reflect"
)

// Field represents a database column mapped from a struct field
type Field struct {
	Name   string       // Struct field name
	Column     string       // DB column name
	Type       reflect.Type // Field type
	Index      int          // Struct field index for fast access
	IsPK       bool         // Is primary key
	IsAuto     bool         // Is auto-increment
	AutoTime   bool         // Set time on insert
	AutoUpdate bool         // Set time on update
	Tag        string       // Raw tag string
}
