package model

import (
	"fmt"
	"reflect"
	"sync"
	"time"
	"unicode"
)

// Model represents table metadata
type Model struct {
	TableName       string
	Fields          []*Field
	FieldMap        map[string]*Field
	PKField         *Field
	Relations       map[string]*Relation
	OriginalType    reflect.Type
	HasBeforeInsert bool
	HasAfterInsert  bool
	HasBeforeUpdate bool
	HasAfterUpdate  bool
	HasBeforeDelete bool
	HasAfterDelete  bool
	HasAfterFind    bool
}

// GetRelation retrieves a relation by name
func (m *Model) GetRelation(name string) (*Relation, error) {
	return GetRelation(m, name)
}

var (
	beforeInserterType = reflect.TypeOf((*BeforeInserter)(nil)).Elem()
	afterInserterType  = reflect.TypeOf((*AfterInserter)(nil)).Elem()
	beforeUpdaterType  = reflect.TypeOf((*BeforeUpdater)(nil)).Elem()
	afterUpdaterType   = reflect.TypeOf((*AfterUpdater)(nil)).Elem()
	beforeDeleterType  = reflect.TypeOf((*BeforeDeleter)(nil)).Elem()
	afterDeleterType   = reflect.TypeOf((*AfterDeleter)(nil)).Elem()
	afterFinderType    = reflect.TypeOf((*AfterFinder)(nil)).Elem()
)

var modelCache sync.Map

// GetModel returns the model metadata for a given value
func GetModel(value any) (*Model, error) {
	if value == nil {
		return nil, fmt.Errorf("failed to get model: value is nil")
	}

	typ := reflect.TypeOf(value)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("failed to get model for %s: value must be a struct or pointer to struct", typ.Kind())
	}

	if cached, ok := modelCache.Load(typ); ok {
		return cached.(*Model), nil
	}

	m, err := parseModel(typ)
	if err != nil {
		return nil, fmt.Errorf("failed to parse model for %s: %w", typ.Name(), err)
	}

	InvalidateRelationCache()
	modelCache.Store(typ, m)
	return m, nil
}

func parseModel(typ reflect.Type) (*Model, error) {
	tableName := camelToSnake(typ.Name())

	// Check if the type implements TableName() string
	// We need a value to check for method implementation
	val := reflect.New(typ).Interface()
	if tn, ok := val.(interface{ TableName() string }); ok {
		tableName = tn.TableName()
	} else if tn, ok := reflect.New(typ).Elem().Interface().(interface{ TableName() string }); ok {
		// Also check value receiver
		tableName = tn.TableName()
	}

	m := &Model{
		TableName:    tableName,
		FieldMap:     make(map[string]*Field),
		Relations:    make(map[string]*Relation),
		OriginalType: typ,
	}

	ptrType := reflect.PtrTo(typ)
	m.HasBeforeInsert = ptrType.Implements(beforeInserterType)
	m.HasAfterInsert = ptrType.Implements(afterInserterType)
	m.HasBeforeUpdate = ptrType.Implements(beforeUpdaterType)
	m.HasAfterUpdate = ptrType.Implements(afterUpdaterType)
	m.HasBeforeDelete = ptrType.Implements(beforeDeleterType)
	m.HasAfterDelete = ptrType.Implements(afterDeleterType)
	m.HasAfterFind = ptrType.Implements(afterFinderType)

	if err := m.parseFields(typ, nil); err != nil {
		return nil, err
	}

	return m, nil
}

func (m *Model) parseFields(typ reflect.Type, baseIndex []int) error {
	for i := 0; i < typ.NumField(); i++ {
		structField := typ.Field(i)
		if !structField.IsExported() {
			continue
		}

		// Handle embedded structs
		if structField.Anonymous {
			fieldTyp := structField.Type
			for fieldTyp.Kind() == reflect.Ptr {
				fieldTyp = fieldTyp.Elem()
			}
			if fieldTyp.Kind() == reflect.Struct {
				newBaseIndex := make([]int, len(baseIndex), len(baseIndex)+1)
				copy(newBaseIndex, baseIndex)
				newBaseIndex = append(newBaseIndex, i)
				if err := m.parseFields(fieldTyp, newBaseIndex); err != nil {
					return err
				}
				continue
			}
		}

		tagStr := structField.Tag.Get("jorm")
		if tagStr == "-" {
			continue
		}

		tag := ParseTag(tagStr)
		if tag.JoinTable != "" || tag.RelationType != "" {
			continue
		}

		if structField.Type.Kind() == reflect.Slice || structField.Type.Kind() == reflect.Map {
			if structField.Type.Kind() == reflect.Slice && structField.Type.Elem().Kind() == reflect.Uint8 {
				// Allow []byte for blob/binary
			} else {
				continue
			}
		}

		if structField.Type.Kind() == reflect.Ptr {
			elemType := structField.Type.Elem()
			if elemType.Kind() == reflect.Slice || elemType.Kind() == reflect.Map {
				continue
			}

			if elemType.Kind() == reflect.Struct && isRelationField(elemType) {
				continue
			}
		}

		columnName := tag.Column
		if columnName == "" {
			columnName = camelToSnake(structField.Name)
		}

		// Calculate nested index
		index := make([]int, len(baseIndex), len(baseIndex)+1)
		copy(index, baseIndex)
		index = append(index, i)

		field := &Field{
			Name:       structField.Name,
			Column:     columnName,
			Type:       structField.Type,
			Index:      i,
			NestedIdx:  index,
			IsPK:       tag.PrimaryKey,
			IsAuto:     tag.AutoInc,
			AutoTime:   tag.AutoTime,
			AutoUpdate: tag.AutoUpdate,
			IsUnique:   tag.Unique,
			Size:       tag.Size,
			NotNull:    tag.NotNull,
			Default:    tag.Default,
			SQLType:    tag.Type,
			Tag:        tagStr,
		}
		field.Accessor = m.createAccessor(field.NestedIdx)

		if err := validateField(field); err != nil {
			return err
		}

		m.Fields = append(m.Fields, field)
		m.FieldMap[columnName] = field

		if field.IsPK {
			m.PKField = field
		}
	}
	return nil
}

func (m *Model) createAccessor(nestedIdx []int) Accessor {
	return func(dest reflect.Value) reflect.Value {
		f := dest
		for _, i := range nestedIdx {
			if f.Kind() == reflect.Ptr {
				if f.IsNil() {
					if !f.CanSet() {
						return reflect.Value{}
					}
					f.Set(reflect.New(f.Type().Elem()))
				}
				f = f.Elem()
			}
			f = f.Field(i)
		}
		return f
	}
}

func isRelationField(typ reflect.Type) bool {
	if typ.Kind() != reflect.Struct {
		return false
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		tagStr := field.Tag.Get("jorm")
		if tagStr == "-" {
			return true
		}
		tag := ParseTag(tagStr)
		if tag.JoinTable != "" || tag.RelationType != "" {
			return true
		}
		if tag.ForeignKey != "" && !tag.PrimaryKey && !tag.AutoInc {
			return true
		}
	}

	return false
}

func camelToSnake(s string) string {
	if s == "ID" {
		return "id"
	}
	var res []rune
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 && (unicode.IsLower(rune(s[i-1])) || (i+1 < len(s) && unicode.IsLower(rune(s[i+1])))) {
				res = append(res, '_')
			}
			res = append(res, unicode.ToLower(r))
		} else {
			res = append(res, r)
		}
	}
	return string(res)
}

func validateField(f *Field) error {
	// Check AutoTime/AutoUpdate
	if f.AutoTime || f.AutoUpdate {
		t := f.Type
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t != reflect.TypeOf(time.Time{}) {
			return fmt.Errorf("field %s has auto_time/auto_update tag but type is %s (must be time.Time)", f.Name, f.Type)
		}
	}

	// Check IsAuto (Auto Increment)
	if f.IsAuto {
		t := f.Type
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		switch t.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			// OK
		default:
			return fmt.Errorf("field %s has auto tag but type is %s (must be integer)", f.Name, f.Type)
		}
	}
	return nil
}
