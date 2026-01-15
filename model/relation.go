package model

import (
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
)

type RelationType int

const (
	RelationHasMany RelationType = iota
	RelationBelongsTo
	RelationHasOne
	RelationManyToMany
)

type Relation struct {
	Name       string       // 关联名称（字段名）
	Type       RelationType // 关联类型
	Model      *Model       // 关联模型元数据
	ForeignKey string       // 外键字段名
	References string       // 引用字段名
	JoinTable  string       // 多对多中间表名
	JoinFK     string       // 中间表外键（指向主表）
	JoinRef    string       // 中间表引用键（指向关联表）
}

type RelationConfig struct {
	RelationType string
	ForeignKey   string
	References   string
	JoinTable    string
	JoinFK       string
	JoinRef      string
}

type relationWithVersion struct {
	relation *Relation
	version  uint64
}

var (
	relationCache        sync.Map
	relationCacheVersion atomic.Uint64
)

func GetRelation(m *Model, name string) (*Relation, error) {
	key := m.TableName + "." + name
	version := relationCacheVersion.Load()

	if cached, ok := relationCache.Load(key); ok {
		rel := cached.(*relationWithVersion)
		if rel.version == version {
			return rel.relation, nil
		}
	}

	relation, err := parseRelationFromTyp(m.OriginalType, name)
	if err != nil {
		return nil, err
	}

	relationCache.Store(key, &relationWithVersion{
		relation: relation,
		version:  version,
	})
	return relation, nil
}

func parseRelationFromTyp(typ reflect.Type, name string) (*Relation, error) {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	for i := 0; i < typ.NumField(); i++ {
		structField := typ.Field(i)
		if !structField.IsExported() {
			continue
		}

		if structField.Name == name {
			return parseRelationFromField(typ, structField)
		}
	}

	return nil, fmt.Errorf("relation '%s' not found", name)
}

func parseRelationFromField(typ reflect.Type, field reflect.StructField) (*Relation, error) {
	tag := ParseTag(field.Tag.Get("jorm"))

	relationType, err := parseRelationType(tag.RelationType, field.Type, tag)
	if err != nil {
		return nil, err
	}

	relation := &Relation{
		Name:  field.Name,
		Type:  relationType,
		Model: nil,
	}

	switch relationType {
	case RelationHasMany, RelationHasOne:
		if tag.ForeignKey == "" {
			tag.ForeignKey = camelToSnake(typ.Name()) + "_id"
		}
		relation.ForeignKey = tag.ForeignKey
		if tag.References == "" {
			relation.References = "id"
		} else {
			relation.References = tag.References
		}

	case RelationBelongsTo:
		if tag.ForeignKey == "" {
			tag.ForeignKey = camelToSnake(field.Name) + "_id"
		}
		relation.ForeignKey = tag.ForeignKey
		if tag.References == "" {
			relation.References = "id"
		} else {
			relation.References = tag.References
		}

	case RelationManyToMany:
		if tag.JoinTable == "" {
			return nil, fmt.Errorf("many_to_many relation requires join_table tag")
		}
		relation.JoinTable = tag.JoinTable

		if tag.JoinFK == "" {
			tag.JoinFK = camelToSnake(typ.Name()) + "_id"
		}
		relation.JoinFK = tag.JoinFK

		if tag.JoinRef == "" {
			// Try to guess JoinRef from the field type (slice element type)
			elemType := field.Type
			if elemType.Kind() == reflect.Slice {
				elemType = elemType.Elem()
			}
			if elemType.Kind() == reflect.Ptr {
				elemType = elemType.Elem()
			}
			relation.JoinRef = camelToSnake(elemType.Name()) + "_id"
		} else {
			relation.JoinRef = tag.JoinRef
		}
	}

	return relation, nil
}

func parseRelation(m *Model, field *Field) (*Relation, error) {
	tag := ParseTag(field.Tag)

	relationType, err := parseRelationType(tag.RelationType, field.Type, tag)
	if err != nil {
		return nil, err
	}

	relation := &Relation{
		Name:  field.Name,
		Type:  relationType,
		Model: nil,
	}

	switch relationType {
	case RelationHasMany, RelationHasOne:
		if tag.ForeignKey == "" {
			tag.ForeignKey = m.TableName + "_id"
		}
		relation.ForeignKey = tag.ForeignKey
		if tag.References == "" {
			if m.PKField != nil {
				relation.References = m.PKField.Column
			} else {
				relation.References = "id"
			}
		} else {
			relation.References = tag.References
		}

	case RelationBelongsTo:
		if tag.ForeignKey == "" {
			tag.ForeignKey = camelToSnake(field.Name) + "_id"
		}
		relation.ForeignKey = tag.ForeignKey
		if tag.References == "" {
			relation.References = "id"
		} else {
			relation.References = tag.References
		}

	case RelationManyToMany:
		if tag.JoinTable == "" {
			return nil, fmt.Errorf("many_to_many relation requires join_table tag")
		}
		relation.JoinTable = tag.JoinTable

		if tag.JoinFK == "" {
			tag.JoinFK = m.TableName + "_id"
		}
		relation.JoinFK = tag.JoinFK

		if tag.JoinRef == "" {
			// Try to guess JoinRef from the field type
			elemType := field.Type
			if elemType.Kind() == reflect.Slice {
				elemType = elemType.Elem()
			}
			if elemType.Kind() == reflect.Ptr {
				elemType = elemType.Elem()
			}
			relation.JoinRef = camelToSnake(elemType.Name()) + "_id"
		} else {
			relation.JoinRef = tag.JoinRef
		}
	}

	return relation, nil
}

func parseRelationType(relationType string, typ reflect.Type, tag *Tag) (RelationType, error) {
	if relationType != "" {
		switch relationType {
		case "has_many":
			return RelationHasMany, nil
		case "belongs_to":
			return RelationBelongsTo, nil
		case "has_one":
			return RelationHasOne, nil
		case "many_to_many":
			return RelationManyToMany, nil
		default:
			return 0, fmt.Errorf("unknown relation type: %s", relationType)
		}
	}

	if tag.JoinTable != "" {
		return RelationManyToMany, nil
	}

	if typ.Kind() == reflect.Slice {
		return RelationHasMany, nil
	}

	return RelationBelongsTo, nil
}

func InvalidateRelationCache() {
	relationCacheVersion.Add(1)
}
