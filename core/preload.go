package core

import (
	"context"
	"database/sql"
	"reflect"
	"sync"

	"github.com/shrek82/jorm/model"
)

type preloadConfig struct {
	path    []string
	builder func(*Query)
}

type preloadExecutor struct {
	db       *DB
	executor Executor
	ctx      context.Context
}

var preloadExecutorPool = sync.Pool{
	New: func() any {
		return &preloadExecutor{}
	},
}

func getPreloadExecutor(db *DB, executor Executor, ctx context.Context) *preloadExecutor {
	exec := preloadExecutorPool.Get().(*preloadExecutor)
	exec.db = db
	exec.executor = executor
	exec.ctx = ctx
	return exec
}

func putPreloadExecutor(exec *preloadExecutor) {
	preloadExecutorPool.Put(exec)
}

func (q *Query) executePreloads(dest any) error {
	if len(q.preloads) == 0 {
		return nil
	}

	exec := getPreloadExecutor(q.db, q.executor, q.ctx)
	defer putPreloadExecutor(exec)

	for _, config := range q.preloads {
		if err := exec.execute(q.model, dest, config); err != nil {
			return err
		}
	}

	return nil
}

func (e *preloadExecutor) execute(mainModel *model.Model, dest any, config *preloadConfig) error {
	if len(config.path) == 0 {
		return nil
	}

	relation, err := mainModel.GetRelation(config.path[0])
	if err != nil {
		return err
	}

	if relation.Model == nil {
		fieldType := getRelationFieldType(mainModel, relation.Name)
		if fieldType == nil {
			return nil
		}
		relModel, err := model.GetModel(reflect.New(fieldType).Interface())
		if err != nil {
			return err
		}
		relation.Model = relModel
	}

	switch relation.Type {
	case model.RelationHasMany, model.RelationHasOne:
		if err := e.executeHasRelation(mainModel, dest, relation, config); err != nil {
			return err
		}
	case model.RelationBelongsTo:
		if err := e.executeBelongsTo(mainModel, dest, relation, config); err != nil {
			return err
		}
	case model.RelationManyToMany:
		if err := e.executeManyToMany(mainModel, dest, relation, config); err != nil {
			return err
		}
	}

	if len(config.path) > 1 {
		nestedConfig := &preloadConfig{
			path:    config.path[1:],
			builder: config.builder,
		}
		return e.executeNested(mainModel, dest, relation, nestedConfig)
	}

	return nil
}

func (e *preloadExecutor) executeHasRelation(mainModel *model.Model, dest any, relation *model.Relation, config *preloadConfig) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return nil
	}

	var sliceValue reflect.Value
	isSlice := destValue.Elem().Kind() == reflect.Slice

	if isSlice {
		sliceValue = destValue.Elem()
	} else {
		// Create a temporary slice for processing
		sliceValue = reflect.MakeSlice(reflect.SliceOf(destValue.Type().Elem()), 1, 1)
		sliceValue.Index(0).Set(destValue.Elem())
	}

	if sliceValue.Len() == 0 {
		return nil
	}

	pkField := mainModel.PKField
	if pkField == nil {
		return nil
	}

	ids, err := e.collectPrimaryKeys(sliceValue, pkField)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return nil
	}

	relatedData, err := e.queryHasRelationData(relation, ids, config)
	if err != nil {
		return err
	}

	if isSlice {
		return e.mapHasRelation(sliceValue, relation, pkField, relatedData)
	} else {
		// Map back to the single object
		err := e.mapHasRelation(sliceValue, relation, pkField, relatedData)
		if err == nil {
			destValue.Elem().Set(sliceValue.Index(0))
		}
		return err
	}
}

func (e *preloadExecutor) executeBelongsTo(mainModel *model.Model, dest any, relation *model.Relation, config *preloadConfig) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return nil
	}

	var sliceValue reflect.Value
	isSlice := destValue.Elem().Kind() == reflect.Slice

	if isSlice {
		sliceValue = destValue.Elem()
	} else {
		sliceValue = reflect.MakeSlice(reflect.SliceOf(destValue.Type().Elem()), 1, 1)
		sliceValue.Index(0).Set(destValue.Elem())
	}

	if sliceValue.Len() == 0 {
		return nil
	}

	var fkField *model.Field
	if field, ok := mainModel.FieldMap[relation.ForeignKey]; ok {
		fkField = field
	} else {
		for _, f := range mainModel.Fields {
			if f.Name == relation.ForeignKey {
				fkField = f
				break
			}
		}
	}

	if fkField == nil {
		return nil
	}

	ids, err := e.collectForeignKeys(sliceValue, fkField)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return nil
	}

	relatedData, err := e.queryBelongsToData(relation, ids, config)
	if err != nil {
		return err
	}

	if isSlice {
		return e.mapBelongsTo(sliceValue, relation, fkField, relatedData)
	} else {
		err := e.mapBelongsTo(sliceValue, relation, fkField, relatedData)
		if err == nil {
			destValue.Elem().Set(sliceValue.Index(0))
		}
		return err
	}
}

func (e *preloadExecutor) executeManyToMany(mainModel *model.Model, dest any, relation *model.Relation, config *preloadConfig) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return nil
	}

	var sliceValue reflect.Value
	isSlice := destValue.Elem().Kind() == reflect.Slice

	if isSlice {
		sliceValue = destValue.Elem()
	} else {
		sliceValue = reflect.MakeSlice(reflect.SliceOf(destValue.Type().Elem()), 1, 1)
		sliceValue.Index(0).Set(destValue.Elem())
	}

	if sliceValue.Len() == 0 {
		return nil
	}

	pkField := mainModel.PKField
	if pkField == nil {
		return nil
	}

	ids, err := e.collectPrimaryKeys(sliceValue, pkField)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return nil
	}

	relatedData, err := e.queryManyToManyData(relation, ids, config)
	if err != nil {
		return err
	}

	if isSlice {
		return e.mapManyToMany(sliceValue, relation, pkField, relatedData)
	} else {
		err := e.mapManyToMany(sliceValue, relation, pkField, relatedData)
		if err == nil {
			destValue.Elem().Set(sliceValue.Index(0))
		}
		return err
	}
}

func (e *preloadExecutor) executeNested(mainModel *model.Model, dest any, relation *model.Relation, config *preloadConfig) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return nil
	}

	sliceValue := destValue.Elem()
	if sliceValue.Len() == 0 {
		return nil
	}

	for i := 0; i < sliceValue.Len(); i++ {
		item := sliceValue.Index(i)
		field := item.FieldByName(relation.Name)

		if field.IsValid() && !field.IsZero() {
			if field.Kind() == reflect.Ptr {
				if field.IsNil() {
					continue
				}
				field = field.Elem()
			}

			if field.Kind() == reflect.Slice && field.Len() > 0 {
				if err := e.execute(relation.Model, field.Addr().Interface(), config); err != nil {
					return err
				}
			} else if field.Kind() == reflect.Struct {
				if err := e.execute(relation.Model, field.Addr().Interface(), config); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (e *preloadExecutor) collectPrimaryKeys(slice reflect.Value, pkField *model.Field) ([]any, error) {
	ids := make([]any, 0, slice.Len())
	for i := 0; i < slice.Len(); i++ {
		item := slice.Index(i)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}
		pkValue := item.Field(pkField.Index).Interface()
		ids = append(ids, pkValue)
	}
	return ids, nil
}

func (e *preloadExecutor) collectForeignKeys(slice reflect.Value, fkField *model.Field) ([]any, error) {
	ids := make([]any, 0, slice.Len())
	for i := 0; i < slice.Len(); i++ {
		item := slice.Index(i)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}
		fkValue := item.Field(fkField.Index).Interface()
		if fkValue != nil && !reflect.ValueOf(fkValue).IsZero() {
			ids = append(ids, fkValue)
		}
	}
	return ids, nil
}

func (e *preloadExecutor) queryHasRelationData(relation *model.Relation, ids []any, config *preloadConfig) (map[any][]any, error) {
	builder := NewBuilder(e.db.dialect)
	builder.Select("*")
	builder.SetTable(relation.Model.TableName)

	columnName := relation.ForeignKey
	if field, ok := relation.Model.FieldMap[columnName]; ok {
		columnName = field.Column
	} else {
		for _, f := range relation.Model.Fields {
			if f.Name == columnName {
				columnName = f.Column
				break
			}
		}
	}
	builder.WhereIn(columnName, ids)

	if config.builder != nil {
		tempQuery := &Query{
			db:       e.db,
			executor: e.executor,
			builder:  builder,
			ctx:      e.ctx,
			model:    relation.Model,
		}
		config.builder(tempQuery)
	}

	sqlStr, args := builder.BuildSelect()
	PutBuilder(builder)

	rows, err := e.executor.QueryContext(e.ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[any][]any)

	for rows.Next() {
		item := reflect.New(relation.Model.OriginalType).Interface()
		if err := e.scanRow(rows, item); err != nil {
			return nil, err
		}

		fkValue := getFieldValue(item, columnName)
		result[fkValue] = append(result[fkValue], item)
	}

	return result, rows.Err()
}

func (e *preloadExecutor) queryBelongsToData(relation *model.Relation, ids []any, config *preloadConfig) (map[any]any, error) {
	builder := NewBuilder(e.db.dialect)
	builder.Select("*")
	builder.SetTable(relation.Model.TableName)

	columnName := relation.References
	if field, ok := relation.Model.FieldMap[columnName]; ok {
		columnName = field.Column
	} else {
		for _, f := range relation.Model.Fields {
			if f.Name == columnName {
				columnName = f.Column
				break
			}
		}
	}
	builder.WhereIn(columnName, ids)

	if config.builder != nil {
		tempQuery := &Query{
			db:       e.db,
			executor: e.executor,
			builder:  builder,
			ctx:      e.ctx,
			model:    relation.Model,
		}
		config.builder(tempQuery)
	}

	sqlStr, args := builder.BuildSelect()
	PutBuilder(builder)

	rows, err := e.executor.QueryContext(e.ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[any]any)

	for rows.Next() {
		item := reflect.New(relation.Model.OriginalType).Interface()
		if err := e.scanRow(rows, item); err != nil {
			return nil, err
		}

		pkValue := getFieldValue(item, columnName)
		result[pkValue] = item
	}

	return result, rows.Err()
}

func (e *preloadExecutor) queryManyToManyData(relation *model.Relation, ids []any, config *preloadConfig) (map[any][]any, error) {
	joinQuery := NewBuilder(e.db.dialect)
	joinQuery.Select("jt."+relation.JoinFK, "jt."+relation.JoinRef)
	joinQuery.SetTable(relation.JoinTable).Alias("jt")
	joinQuery.WhereIn("jt."+relation.JoinFK, ids)

	sqlStr, args := joinQuery.BuildSelect()
	PutBuilder(joinQuery)

	rows, err := e.executor.QueryContext(e.ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fkToRefs := make(map[any][]any)
	for rows.Next() {
		var fkValue, refValue any
		if err := rows.Scan(&fkValue, &refValue); err != nil {
			return nil, err
		}
		fkToRefs[fkValue] = append(fkToRefs[fkValue], refValue)
	}

	allRefValues := make([]any, 0)
	for _, refs := range fkToRefs {
		allRefValues = append(allRefValues, refs...)
	}

	if len(allRefValues) == 0 {
		return make(map[any][]any), nil
	}

	builder := NewBuilder(e.db.dialect)
	builder.Select("*")
	builder.SetTable(relation.Model.TableName)

	pkColumn := relation.Model.PKField.Column
	builder.WhereIn(pkColumn, allRefValues)

	if config.builder != nil {
		tempQuery := &Query{
			db:       e.db,
			executor: e.executor,
			builder:  builder,
			ctx:      e.ctx,
			model:    relation.Model,
		}
		config.builder(tempQuery)
	}

	sqlStr, args = builder.BuildSelect()
	PutBuilder(builder)

	dataRows, err := e.executor.QueryContext(e.ctx, sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer dataRows.Close()

	refToData := make(map[any]any)
	for dataRows.Next() {
		item := reflect.New(relation.Model.OriginalType).Interface()
		if err := e.scanRow(dataRows, item); err != nil {
			return nil, err
		}
		pkValue := getFieldValue(item, pkColumn)
		refToData[pkValue] = item
	}

	result := make(map[any][]any)
	for fk, refs := range fkToRefs {
		for _, ref := range refs {
			if data, ok := refToData[ref]; ok {
				result[fk] = append(result[fk], data)
			}
		}
	}

	return result, nil
}

func (e *preloadExecutor) mapHasRelation(slice reflect.Value, relation *model.Relation, pkField *model.Field, data map[any][]any) error {
	fieldIndex := getRelationFieldIndex(slice.Type().Elem(), relation.Name)
	if fieldIndex < 0 {
		return nil
	}

	for i := 0; i < slice.Len(); i++ {
		item := slice.Index(i)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}
		pkValue := item.Field(pkField.Index).Interface()

		items, ok := data[pkValue]
		if !ok {
			continue
		}

		field := item.Field(fieldIndex)
		if relation.Type == model.RelationHasOne {
			if len(items) > 0 {
				if field.Kind() == reflect.Ptr {
					if !field.IsNil() && !field.IsZero() {
						continue
					}
					field.Set(reflect.ValueOf(items[0]))
				} else {
					field.Set(reflect.ValueOf(items[0]).Elem())
				}
			}
		} else {
			sliceType := field.Type()
			if sliceType.Kind() == reflect.Ptr {
				sliceType = sliceType.Elem()
			}

			newSlice := reflect.MakeSlice(sliceType, 0, len(items))
			for _, item := range items {
				if sliceType.Elem().Kind() == reflect.Ptr {
					newSlice = reflect.Append(newSlice, reflect.ValueOf(item))
				} else {
					newSlice = reflect.Append(newSlice, reflect.ValueOf(item).Elem())
				}
			}
			field.Set(newSlice)
		}
	}

	return nil
}

func (e *preloadExecutor) mapBelongsTo(slice reflect.Value, relation *model.Relation, fkField *model.Field, data map[any]any) error {
	fieldIndex := getRelationFieldIndex(slice.Type().Elem(), relation.Name)
	if fieldIndex < 0 {
		return nil
	}

	for i := 0; i < slice.Len(); i++ {
		item := slice.Index(i)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}
		fkValue := item.Field(fkField.Index).Interface()

		relatedData, ok := data[fkValue]
		if !ok {
			continue
		}

		field := item.Field(fieldIndex)
		if field.Kind() == reflect.Ptr {
			field.Set(reflect.ValueOf(relatedData))
		} else {
			field.Set(reflect.ValueOf(relatedData).Elem())
		}
	}

	return nil
}

func (e *preloadExecutor) mapManyToMany(slice reflect.Value, relation *model.Relation, pkField *model.Field, data map[any][]any) error {
	fieldIndex := getRelationFieldIndex(slice.Type().Elem(), relation.Name)
	if fieldIndex < 0 {
		return nil
	}

	for i := 0; i < slice.Len(); i++ {
		item := slice.Index(i)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}
		pkValue := item.Field(pkField.Index).Interface()

		items, ok := data[pkValue]
		if !ok {
			continue
		}

		field := item.Field(fieldIndex)
		sliceType := field.Type()
		if sliceType.Kind() == reflect.Ptr {
			sliceType = sliceType.Elem()
		}

		newSlice := reflect.MakeSlice(sliceType, 0, len(items))
		for _, item := range items {
			if sliceType.Elem().Kind() == reflect.Ptr {
				newSlice = reflect.Append(newSlice, reflect.ValueOf(item))
			} else {
				newSlice = reflect.Append(newSlice, reflect.ValueOf(item).Elem())
			}
		}
		field.Set(newSlice)
	}

	return nil
}

func (e *preloadExecutor) scanRow(rows *sql.Rows, dest any) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	m, err := model.GetModel(dest)
	if err != nil {
		return err
	}

	plan := getScanPlan(m, columns)

	values := make([]any, len(columns))
	for i, field := range plan.fields {
		if field != nil {
			values[i] = reflect.New(field.Type).Interface()
		} else {
			var ignore any
			values[i] = &ignore
		}
	}

	if err := rows.Scan(values...); err != nil {
		return err
	}

	destValue := reflect.ValueOf(dest).Elem()
	for i, field := range plan.fields {
		if field != nil {
			val := reflect.ValueOf(values[i]).Elem()
			setFieldValue(destValue, field, val, plan, i)
		}
	}

	return nil
}

func getRelationFieldType(m *model.Model, fieldName string) reflect.Type {
	if field, ok := m.OriginalType.FieldByName(fieldName); ok {
		t := field.Type
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t.Kind() == reflect.Slice {
			t = t.Elem()
			for t.Kind() == reflect.Ptr {
				t = t.Elem()
			}
		}
		return t
	}
	return nil
}

func relationModelElementType(m *model.Model) reflect.Type {
	return m.OriginalType
}

func getFieldValue(item any, columnName string) any {
	v := reflect.ValueOf(item)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	m, err := model.GetModel(item)
	if err != nil {
		return nil
	}

	if field, ok := m.FieldMap[columnName]; ok {
		if len(field.NestedIdx) > 0 {
			return v.FieldByIndex(field.NestedIdx).Interface()
		}
		return v.Field(field.Index).Interface()
	}

	// Try searching by name if column name doesn't match
	for _, field := range m.Fields {
		if field.Column == columnName {
			if len(field.NestedIdx) > 0 {
				return v.FieldByIndex(field.NestedIdx).Interface()
			}
			return v.Field(field.Index).Interface()
		}
	}

	return nil
}

func getRelationFieldIndex(typ reflect.Type, fieldName string) int {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return -1
	}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Name == fieldName {
			return i
		}
	}
	return -1
}
