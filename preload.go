package gorm

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

// Preload preload relations callback
func Preload(scope *Scope) {
	if scope.Search.preload == nil || scope.HasError() {
		return
	}

	var (
		preloadedMap = map[string]bool{}
		fields       = scope.Fields()
	)

	for _, preload := range scope.Search.preload {
		var (
			preloadFields = strings.Split(preload.schema, ".")
			currentScope  = scope
			currentFields = fields
		)

		for idx, preloadField := range preloadFields {
			var currentPreloadConditions []interface{}

			// if not preloaded
			if preloadKey := strings.Join(preloadFields[:idx+1], "."); !preloadedMap[preloadKey] {

				// assign search conditions to last preload
				if idx == len(preloadFields)-1 {
					currentPreloadConditions = preload.conditions
				}

				for _, field := range currentFields {
					if field.Name != preloadField || field.Relationship == nil {
						continue
					}

					switch field.Relationship.Kind {
					case "has_one":
						currentScope.handleHasOnePreload(field, currentPreloadConditions)
					case "has_many":
						currentScope.handleHasManyPreload(field, currentPreloadConditions)
					case "belongs_to":
						currentScope.handleBelongsToPreload(field, currentPreloadConditions)
					case "many_to_many":
						currentScope.handleManyToManyPreload(field, currentPreloadConditions)
					default:
						scope.Err(errors.New("unsupported relation"))
					}

					preloadedMap[preloadKey] = true
					break
				}

				if !preloadedMap[preloadKey] {
					scope.Err(fmt.Errorf("can't preload field %s for %s", preloadField, currentScope.GetModelStruct().ModelType))
					return
				}
			}

			// preload next level
			if idx < len(preloadFields)-1 {
				currentScope = currentScope.getColumnAsScope(preloadField)
				currentFields = currentScope.Fields()
			}
		}
	}
}

func (scope *Scope) handleHasOnePreload(field *Field, conditions []interface{}) {
	relation := field.Relationship

	// get relations's primary keys
	primaryKeys := scope.getColumnAsArray(relation.AssociationForeignFieldNames, scope.Value)
	if len(primaryKeys) == 0 {
		return
	}

	// find relations
	results := makeSlice(field.Struct.Type)
	scope.Err(scope.NewDB().Where(fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, relation.ForeignDBNames), toQueryMarks(primaryKeys)), toQueryValues(primaryKeys)...).Find(results, conditions...).Error)

	// assign find results
	var (
		resultsValue       = reflect.Indirect(reflect.ValueOf(results))
		indirectScopeValue = scope.IndirectValue()
	)

	for i := 0; i < resultsValue.Len(); i++ {
		result := resultsValue.Index(i)
		if indirectScopeValue.Kind() == reflect.Slice {
			foreignValues := getValueFromFields(result, relation.ForeignFieldNames)
			for j := 0; j < indirectScopeValue.Len(); j++ {
				if indirectValue := reflect.Indirect(indirectScopeValue.Index(j)); equalAsString(getValueFromFields(indirectValue, relation.AssociationForeignFieldNames), foreignValues) {
					indirectValue.FieldByName(field.Name).Set(result)
					break
				}
			}
		} else {
			scope.Err(field.Set(result))
		}
	}
}

func (scope *Scope) handleHasManyPreload(field *Field, conditions []interface{}) {
	relation := field.Relationship

	// get relations's primary keys
	primaryKeys := scope.getColumnAsArray(relation.AssociationForeignFieldNames, scope.Value)
	if len(primaryKeys) == 0 {
		return
	}

	// find relations
	results := makeSlice(field.Struct.Type)
	scope.Err(scope.NewDB().Where(fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, relation.ForeignDBNames), toQueryMarks(primaryKeys)), toQueryValues(primaryKeys)...).Find(results, conditions...).Error)

	// assign find results
	var (
		resultsValue       = reflect.Indirect(reflect.ValueOf(results))
		indirectScopeValue = scope.IndirectValue()
	)

	if indirectScopeValue.Kind() == reflect.Slice {
		for i := 0; i < resultsValue.Len(); i++ {
			result := resultsValue.Index(i)
			foreignValues := getValueFromFields(result, relation.ForeignFieldNames)
			for j := 0; j < indirectScopeValue.Len(); j++ {
				object := reflect.Indirect(indirectScopeValue.Index(j))
				if equalAsString(getValueFromFields(object, relation.AssociationForeignFieldNames), foreignValues) {
					objectField := object.FieldByName(field.Name)
					objectField.Set(reflect.Append(objectField, result))
					break
				}
			}
		}
	} else {
		scope.Err(field.Set(resultsValue))
	}
}

func (scope *Scope) handleBelongsToPreload(field *Field, conditions []interface{}) {
	relation := field.Relationship

	// get relations's primary keys
	primaryKeys := scope.getColumnAsArray(relation.ForeignFieldNames, scope.Value)
	if len(primaryKeys) == 0 {
		return
	}

	// find relations
	results := makeSlice(field.Struct.Type)
	scope.Err(scope.NewDB().Where(fmt.Sprintf("%v IN (%v)", toQueryCondition(scope, relation.AssociationForeignDBNames), toQueryMarks(primaryKeys)), toQueryValues(primaryKeys)...).Find(results, conditions...).Error)

	// assign find results
	var (
		resultsValue       = reflect.Indirect(reflect.ValueOf(results))
		indirectScopeValue = scope.IndirectValue()
	)

	for i := 0; i < resultsValue.Len(); i++ {
		result := resultsValue.Index(i)
		if indirectScopeValue.Kind() == reflect.Slice {
			value := getValueFromFields(result, relation.AssociationForeignFieldNames)
			for j := 0; j < indirectScopeValue.Len(); j++ {
				object := reflect.Indirect(indirectScopeValue.Index(j))
				if equalAsString(getValueFromFields(object, relation.ForeignFieldNames), value) {
					object.FieldByName(field.Name).Set(result)
				}
			}
		} else {
			scope.Err(field.Set(result))
		}
	}
}

func (scope *Scope) handleManyToManyPreload(field *Field, conditions []interface{}) {
	var (
		relation         = field.Relationship
		joinTableHandler = relation.JoinTableHandler
		fieldType        = field.StructField.Struct.Type.Elem()
		foreignKeyValue  interface{}
		foreignKeyType   = reflect.ValueOf(&foreignKeyValue).Type()
		linkHash         = map[string][]reflect.Value{}
		isPtr            bool
	)

	if fieldType.Kind() == reflect.Ptr {
		isPtr = true
		fieldType = fieldType.Elem()
	}

	var sourceKeys = []string{}
	for _, key := range joinTableHandler.SourceForeignKeys() {
		sourceKeys = append(sourceKeys, key.DBName)
	}

	// generate query with join table
	preloadJoinDB := scope.NewDB().Table(scope.New(reflect.New(fieldType).Interface()).TableName()).Select("*")
	preloadJoinDB = joinTableHandler.JoinWith(joinTableHandler, preloadJoinDB, scope.Value)

	// preload inline conditions
	if len(conditions) > 0 {
		preloadJoinDB = preloadJoinDB.Where(conditions[0], conditions[1:]...)
	}

	rows, err := preloadJoinDB.Rows()

	if scope.Err(err) != nil {
		return
	}
	defer rows.Close()

	columns, _ := rows.Columns()
	for rows.Next() {
		var (
			elem   = reflect.New(fieldType).Elem()
			fields = scope.New(elem.Addr().Interface()).Fields()
		)

		// register foreign keys in join tables
		for _, sourceKey := range sourceKeys {
			fields[sourceKey] = &Field{Field: reflect.New(foreignKeyType).Elem()}
		}

		scope.scan(rows, columns, fields)

		// generate hashed forkey keys in join table
		var foreignKeys = make([]interface{}, len(sourceKeys))
		for idx, sourceKey := range sourceKeys {
			foreignKeys[idx] = fields[sourceKey].Field.Elem().Interface()
		}
		hashedSourceKeys := toString(foreignKeys)

		if isPtr {
			linkHash[hashedSourceKeys] = append(linkHash[hashedSourceKeys], elem.Addr())
		} else {
			linkHash[hashedSourceKeys] = append(linkHash[hashedSourceKeys], elem)
		}
	}

	// assign find results
	var (
		indirectScopeValue = scope.IndirectValue()
		fieldsSourceMap    = map[string]reflect.Value{}
		foreignFieldNames  = []string{}
		fields             = scope.Fields()
	)

	for _, dbName := range relation.ForeignFieldNames {
		if field, ok := fields[dbName]; ok {
			foreignFieldNames = append(foreignFieldNames, field.Name)
		}
	}

	if indirectScopeValue.Kind() == reflect.Slice {
		for j := 0; j < indirectScopeValue.Len(); j++ {
			object := reflect.Indirect(indirectScopeValue.Index(j))
			fieldsSourceMap[toString(getValueFromFields(object, foreignFieldNames))] = object.FieldByName(field.Name)
		}
	} else if indirectScopeValue.IsValid() {
		fieldsSourceMap[toString(getValueFromFields(indirectScopeValue, foreignFieldNames))] = indirectScopeValue.FieldByName(field.Name)
	}

	for source, link := range linkHash {
		fieldsSourceMap[source].Set(reflect.Append(fieldsSourceMap[source], link...))
	}
}
