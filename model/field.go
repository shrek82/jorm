package model

import (
	"reflect"
)

// Accessor is a pre-generated function to access a field value from a struct
type Accessor func(reflect.Value) reflect.Value

// Field represents a database column mapped from a struct field
type Field struct {
	Name       string       // Struct field name
	Column     string       // DB column name
	Type       reflect.Type // Field type
	Index      int          // Struct field index for fast access
	NestedIdx  []int        // Nested field index for embedded structs
	IsPK       bool         // Is primary key
	IsAuto     bool         // Is auto-increment
	AutoTime   bool         // Set time on insert
	AutoUpdate bool         // Set time on update
	IsUnique   bool         // Is unique index
	Tag        string       // Raw tag string
	Accessor   Accessor     // Pre-generated field accessor
}
