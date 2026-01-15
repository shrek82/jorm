package model

import (
	"fmt"
	"reflect"
	"sync"
	"unicode"
)

// Model represents table metadata
type Model struct {
	TableName    string
	Fields       []*Field
	FieldMap     map[string]*Field
	PKField      *Field
	Relations    map[string]*Relation
	OriginalType reflect.Type
}

// GetRelation retrieves a relation by name
func (m *Model) GetRelation(name string) (*Relation, error) {
	return GetRelation(m, name)
}

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

	key := typ.PkgPath() + "." + typ.Name()
	if cached, ok := modelCache.Load(key); ok {
		return cached.(*Model), nil
	}

	m, err := parseModel(typ)
	if err != nil {
		return nil, fmt.Errorf("failed to parse model for %s: %w", typ.Name(), err)
	}

	InvalidateRelationCache()
	modelCache.Store(key, m)
	return m, nil
}

func parseModel(typ reflect.Type) (*Model, error) {
	m := &Model{
		TableName:    camelToSnake(typ.Name()),
		FieldMap:     make(map[string]*Field),
		Relations:    make(map[string]*Relation),
		OriginalType: typ,
	}

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
			continue
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
			Tag:        tagStr,
		}

		m.Fields = append(m.Fields, field)
		m.FieldMap[columnName] = field

		if field.IsPK {
			m.PKField = field
		}
	}
	return nil
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
