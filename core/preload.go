package core

import (
	"context"
	"database/sql"
	"reflect"
	"sync"
	"time"

	"github.com/shrek82/jorm/model"
)

// preloadConfig holds the configuration for a preload operation.
// It includes the path of the relation to preload and an optional query builder function.
type preloadConfig struct {
	// path is the split path of the relation (e.g., ["Orders", "Items"] for "Orders.Items").
	path []string
	// builder is an optional function to customize the query (e.g., adding WHERE clauses).
	builder func(*Query)
}

// preloadExecutor handles the execution of loading related data.
// It manages the lifecycle of a preload operation, including database connection and context.
type preloadExecutor struct {
	// db is the database connection to use for querying related data.
	db *DB
	// executor is the SQL executor (DB or Tx) to ensure transactional consistency.
	executor Executor
	// ctx is the context for the query execution, supporting cancellation and timeouts.
	ctx context.Context
}

// preloadExecutorPool is a pool of preloadExecutor objects to reduce allocation overhead.
// Reusing executors helps minimize garbage collection pressure during high-throughput operations.
var preloadExecutorPool = sync.Pool{
	New: func() any {
		return &preloadExecutor{}
	},
}

// getPreloadExecutor retrieves a preloadExecutor from the pool and initializes it.
// It sets the db, executor, and context for the current operation.
func getPreloadExecutor(db *DB, executor Executor, ctx context.Context) *preloadExecutor {
	exec := preloadExecutorPool.Get().(*preloadExecutor)
	exec.db = db
	exec.executor = executor
	exec.ctx = ctx
	return exec
}

// putPreloadExecutor returns a preloadExecutor to the pool.
// It should be called using defer to ensure the executor is always returned.
func putPreloadExecutor(exec *preloadExecutor) {
	// Reset fields to avoid memory leaks or retaining context
	exec.db = nil
	exec.executor = nil
	exec.ctx = nil
	preloadExecutorPool.Put(exec)
}

// executePreloads executes all registered preload operations for the query.
// It iterates over the preloads configuration and executes them one by one.
// This is the entry point for the preloading mechanism.
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

// execute determines the relation type and delegates to the appropriate handler.
// It also handles nested preloads by recursively calling itself.
//
// Parameters:
//   - mainModel: The model definition of the parent object.
//   - dest: The destination object (slice or pointer to struct) containing the parent data.
//   - config: The configuration for the current preload level.
func (e *preloadExecutor) execute(mainModel *model.Model, dest any, config *preloadConfig) error {
	if len(config.path) == 0 {
		return nil
	}

	// Get the relation definition from the model based on the first path segment
	relation, err := mainModel.GetRelation(config.path[0])
	if err != nil {
		return err
	}

	// Lazy load the related model if it's not already loaded.
	// This is necessary because models might be defined with circular references
	// or loaded in an order where the related model isn't fully initialized yet.
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

	// Switch on relation type to call specific execution logic
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

	// Handle nested preloads (e.g., "Orders.Items")
	// If the path has more segments, we create a new config for the next level
	// and call executeNested to handle the recursion.
	if len(config.path) > 1 {
		nestedConfig := &preloadConfig{
			path:    config.path[1:],
			builder: config.builder,
		}
		return e.executeNested(mainModel, dest, relation, nestedConfig)
	}

	return nil
}

// executeHasRelation handles HasOne and HasMany relations.
// It collects primary keys from the parent objects, queries the related objects,
// and then maps them back to the parents.
//
// Workflow:
// 1. Normalize dest to a slice (handling single object vs slice).
// 2. Collect primary keys (IDs) from parent objects.
// 3. Query related table where Foreign Key IN (IDs).
// 4. Map the results back to the parent objects' fields.
func (e *preloadExecutor) executeHasRelation(mainModel *model.Model, dest any, relation *model.Relation, config *preloadConfig) error {
	destValue := reflect.ValueOf(dest)
	if destValue.Kind() != reflect.Ptr {
		return nil
	}

	var sliceValue reflect.Value
	isSlice := destValue.Elem().Kind() == reflect.Slice

	// Normalize destination to a slice for uniform processing
	if isSlice {
		sliceValue = destValue.Elem()
	} else {
		// Create a temporary slice for processing single object
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

	// Collect IDs from the parent objects
	ids, err := e.collectPrimaryKeys(sliceValue, pkField)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return nil
	}

	// Query related data
	relatedData, err := e.queryHasRelationData(relation, ids, config)
	if err != nil {
		return err
	}

	// Assign related data back to parent objects
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

// executeBelongsTo handles BelongsTo relations.
// It collects foreign keys from the parent objects, queries the related objects,
// and maps them back.
//
// Workflow:
// 1. Normalize dest to a slice.
// 2. Collect Foreign Keys from parent objects.
// 3. Query related table where Primary Key IN (Foreign Keys).
// 4. Map the results back to the parent objects.
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

	// Find the foreign key field in the parent model
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

	// Collect foreign keys
	ids, err := e.collectForeignKeys(sliceValue, fkField)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return nil
	}

	// Query related data
	relatedData, err := e.queryBelongsToData(relation, ids, config)
	if err != nil {
		return err
	}

	// Assign related data back
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

// executeManyToMany handles ManyToMany relations.
// It uses the join table to find associations between parent and child objects.
//
// Workflow:
// 1. Normalize dest to a slice.
// 2. Collect Primary Keys from parent objects.
// 3. Query Join Table to map Parent IDs -> Related IDs.
// 4. Query Related Table where Primary Key IN (Related IDs).
// 5. Map the results back to the parent objects using the mapping from Step 3.
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

	// Collect parent IDs
	ids, err := e.collectPrimaryKeys(sliceValue, pkField)
	if err != nil {
		return err
	}

	if len(ids) == 0 {
		return nil
	}

	// Query related data through join table
	relatedData, err := e.queryManyToManyData(relation, ids, config)
	if err != nil {
		return err
	}

	// Assign back
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

// executeNested handles nested preloading (e.g. loading "Items" for each "Order" in "User.Orders").
// It iterates through the loaded related objects and triggers the next level of preloading for them.
func (e *preloadExecutor) executeNested(mainModel *model.Model, dest any, relation *model.Relation, config *preloadConfig) error {
	destValue := reflect.ValueOf(dest)
	// We expect a slice of parents here (which might have been single objects wrapped in slice previously,
	// but execute() calls us with the original dest if we are not careful.
	// Actually execute() calls us with the SAME dest.
	// So we need to handle both slice and ptr to slice.
	// But wait, the logic below expects sliceValue to be the collection of parents.
	// In execute(), dest is passed through.

	if destValue.Kind() != reflect.Ptr || destValue.Elem().Kind() != reflect.Slice {
		return nil
	}

	sliceValue := destValue.Elem()
	if sliceValue.Len() == 0 {
		return nil
	}

	// For each parent object in the slice
	for i := 0; i < sliceValue.Len(); i++ {
		item := sliceValue.Index(i)
		// Get the field that holds the related data (which we just populated)
		field := item.FieldByName(relation.Name)

		if field.IsValid() && !field.IsZero() {
			if field.Kind() == reflect.Ptr {
				if field.IsNil() {
					continue
				}
				field = field.Elem()
			}

			// Recursively call execute for the related objects
			if field.Kind() == reflect.Slice && field.Len() > 0 {
				// field.Addr().Interface() gives us a pointer to the slice of related objects
				if err := e.execute(relation.Model, field.Addr().Interface(), config); err != nil {
					return err
				}
			} else if field.Kind() == reflect.Struct {
				// Pointer to the related struct
				if err := e.execute(relation.Model, field.Addr().Interface(), config); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// collectPrimaryKeys extracts primary key values from a slice of structs.
// It uses reflection to access the primary key field defined in the model.
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

// collectForeignKeys extracts foreign key values from a slice of structs.
// It filters out zero values/nil to avoid querying with invalid IDs.
func (e *preloadExecutor) collectForeignKeys(slice reflect.Value, fkField *model.Field) ([]any, error) {
	ids := make([]any, 0, slice.Len())
	for i := 0; i < slice.Len(); i++ {
		item := slice.Index(i)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}
		fkValue := item.Field(fkField.Index).Interface()
		// Only collect non-zero FKs
		if fkValue != nil && !reflect.ValueOf(fkValue).IsZero() {
			ids = append(ids, fkValue)
		}
	}
	return ids, nil
}

// queryHasRelationData queries the database for HasOne/HasMany relations.
// Returns a map of ParentID -> Slice of Related Objects.
// It constructs a query selecting all related objects where the foreign key matches the parent IDs.
func (e *preloadExecutor) queryHasRelationData(relation *model.Relation, ids []any, config *preloadConfig) (map[any][]any, error) {
	builder := NewBuilder(e.db.dialect)
	builder.Select("*")
	builder.SetTable(relation.Model.TableName)

	// Determine the column name for the foreign key in the related table
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

	// Apply custom query modifications if provided
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

		// Group by Foreign Key (which matches Parent's Primary Key)
		fkValue := getFieldValue(item, columnName)
		result[fkValue] = append(result[fkValue], item)
	}

	return result, rows.Err()
}

// queryBelongsToData queries the database for BelongsTo relations.
// Returns a map of RelatedID -> Related Object.
// It constructs a query selecting all related objects where the Primary Key matches the Foreign Keys collected from parents.
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

// queryManyToManyData queries the database for ManyToMany relations.
// It involves a join query to map Parent IDs to Related Object IDs.
// Returns a map of ParentID -> Slice of Related Objects.
func (e *preloadExecutor) queryManyToManyData(relation *model.Relation, ids []any, config *preloadConfig) (map[any][]any, error) {
	// Step 1: Query the Join Table to get (ParentID, RelatedID) pairs
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

	// Step 2: Query the Related Table using the collected Related IDs
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

	// Step 3: Reconstruct the result map (ParentID -> Related Objects)
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

// mapHasRelation assigns the loaded HasOne/HasMany data back to the parent objects.
// It matches parent objects with related data using the primary key.
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
					field.Set(reflect.ValueOf(items[0]))
				} else {
					field.Set(reflect.ValueOf(items[0]).Elem())
				}
			}
		} else {
			// HasMany: Assign slice
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

// mapBelongsTo assigns the loaded BelongsTo data back to the parent objects.
// It matches parent objects with related data using the foreign key.
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

// mapManyToMany assigns the loaded ManyToMany data back to the parent objects.
// It matches parent objects with related data using the primary key.
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

// scanRow scans a single row into a destination struct.
// It handles time.Time and pointer fields correctly using the model's scan plan.
// It uses a TimeScanner to handle potential NULL values for time fields.
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
			if field.Type == timeType {
				values[i] = &TimeScanner{}
			} else if field.Type == timePtrType {
				values[i] = &TimeScanner{}
			} else {
				values[i] = reflect.New(field.Type).Interface()
			}
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
			var val reflect.Value
			if ts, ok := values[i].(*TimeScanner); ok {
				if field.Type == timeType {
					if ts.Valid {
						val = reflect.ValueOf(ts.Value)
					} else {
						val = reflect.ValueOf(time.Time{})
					}
				} else { // *time.Time
					if ts.Valid {
						t := ts.Value
						val = reflect.ValueOf(&t)
					} else {
						val = reflect.Zero(field.Type)
					}
				}
			} else {
				val = reflect.ValueOf(values[i]).Elem()
			}
			setFieldValue(destValue, field, val, plan, i)
		}
	}

	return nil
}

// getRelationFieldType resolves the reflection type of a field by name.
// It handles pointers and slices to find the underlying struct type.
// This is used when the relation model is not yet loaded or initialized.
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

// getFieldValue retrieves the value of a field based on the column name.
// It maps the column name to the field name using the model definition.
func getFieldValue(item any, columnName string) any {
	m, err := model.GetModel(item)
	if err != nil {
		return nil
	}

	val := reflect.ValueOf(item)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	for _, field := range m.Fields {
		if field.Column == columnName {
			return val.Field(field.Index).Interface()
		}
	}

	return nil
}

// getRelationFieldIndex returns the index of the field with the given name.
// It assumes the field is a direct member of the struct.
func getRelationFieldIndex(typ reflect.Type, fieldName string) int {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if f, ok := typ.FieldByName(fieldName); ok {
		if len(f.Index) > 0 {
			return f.Index[0]
		}
	}
	return -1
}
