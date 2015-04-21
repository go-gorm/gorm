package gorm

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

func getRealValue(value reflect.Value, field string) interface{} {
	result := reflect.Indirect(value).FieldByName(field).Interface()
	if r, ok := result.(driver.Valuer); ok {
		result, _ = r.Value()
	}
	return result
}

func equalAsString(a interface{}, b interface{}) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func Preload(scope *Scope) {
	if scope.Search.preload == nil {
		return
	}

	preloadMap := map[string]bool{}
	fields := scope.Fields()
	for _, preload := range scope.Search.preload {
		schema, conditions := preload.schema, preload.conditions
		keys := strings.Split(schema, ".")
		currentScope := scope
		currentFields := fields
		originalConditions := conditions
		conditions = []interface{}{}
		for i, key := range keys {
			var found bool
			if preloadMap[strings.Join(keys[:i+1], ".")] {
				goto nextLoop
			}

			if i == len(keys)-1 {
				conditions = originalConditions
			}

			for _, field := range currentFields {
				if field.Name != key || field.Relationship == nil {
					continue
				}

				found = true
				switch field.Relationship.Kind {
				case "has_one":
					currentScope.handleHasOnePreload(field, conditions)
				case "has_many":
					currentScope.handleHasManyPreload(field, conditions)
				case "belongs_to":
					currentScope.handleBelongsToPreload(field, conditions)
				case "many_to_many":
					fallthrough
				default:
					currentScope.Err(errors.New("not supported relation"))
				}
				break
			}

			if !found {
				value := reflect.ValueOf(currentScope.Value)
				if value.Kind() == reflect.Slice && value.Type().Elem().Kind() == reflect.Interface {
					value = value.Index(0).Elem()
				}
				scope.Err(fmt.Errorf("can't found field %s in %s", key, value.Type()))
				return
			}

			preloadMap[strings.Join(keys[:i+1], ".")] = true

		nextLoop:
			if i < len(keys)-1 {
				currentScope = currentScope.getColumnsAsScope(key)
				currentFields = currentScope.Fields()
			}
		}
	}

}

func makeSlice(typ reflect.Type) interface{} {
	if typ.Kind() == reflect.Slice {
		typ = typ.Elem()
	}
	sliceType := reflect.SliceOf(typ)
	slice := reflect.New(sliceType)
	slice.Elem().Set(reflect.MakeSlice(sliceType, 0, 0))
	return slice.Interface()
}

func (scope *Scope) handleHasOnePreload(field *Field, conditions []interface{}) {
	primaryName := scope.PrimaryField().Name
	primaryKeys := scope.getColumnAsArray(primaryName)
	if len(primaryKeys) == 0 {
		return
	}

	results := makeSlice(field.Struct.Type)
	relation := field.Relationship
	condition := fmt.Sprintf("%v IN (?)", scope.Quote(relation.ForeignDBName))
	resultValues := reflect.Indirect(reflect.ValueOf(results))

	// TODO: handle error?
	scope.NewDB().Where(condition, primaryKeys).Find(results, conditions...)

	for i := 0; i < resultValues.Len(); i++ {
		result := resultValues.Index(i)
		if scope.IndirectValue().Kind() == reflect.Slice {
			value := getRealValue(result, relation.ForeignFieldName)
			objects := scope.IndirectValue()
			for j := 0; j < objects.Len(); j++ {
				if equalAsString(getRealValue(objects.Index(j), primaryName), value) {
					reflect.Indirect(objects.Index(j)).FieldByName(field.Name).Set(result)
					break
				}
			}
		} else {
			err := scope.SetColumn(field, result)
			if err != nil {
				scope.Err(err)
				return
			}
		}
	}
}

func (scope *Scope) handleHasManyPreload(field *Field, conditions []interface{}) {
	primaryName := scope.PrimaryField().Name
	primaryKeys := scope.getColumnAsArray(primaryName)
	if len(primaryKeys) == 0 {
		return
	}

	results := makeSlice(field.Struct.Type)
	relation := field.Relationship
	condition := fmt.Sprintf("%v IN (?)", scope.Quote(relation.ForeignDBName))
	resultValues := reflect.Indirect(reflect.ValueOf(results))

	scope.NewDB().Where(condition, primaryKeys).Find(results, conditions...)

	if scope.IndirectValue().Kind() == reflect.Slice {
		for i := 0; i < resultValues.Len(); i++ {
			result := resultValues.Index(i)
			value := getRealValue(result, relation.ForeignFieldName)
			objects := scope.IndirectValue()
			for j := 0; j < objects.Len(); j++ {
				object := reflect.Indirect(objects.Index(j))
				if equalAsString(getRealValue(object, primaryName), value) {
					f := object.FieldByName(field.Name)
					f.Set(reflect.Append(f, result))
					break
				}
			}
		}
	} else {
		scope.SetColumn(field, resultValues)
	}
}

func (scope *Scope) handleBelongsToPreload(field *Field, conditions []interface{}) {
	relation := field.Relationship
	primaryKeys := scope.getColumnAsArray(relation.ForeignFieldName)
	if len(primaryKeys) == 0 {
		return
	}

	results := makeSlice(field.Struct.Type)
	resultValues := reflect.Indirect(reflect.ValueOf(results))
	associationPrimaryKey := scope.New(results).PrimaryField().Name

	scope.NewDB().Where(primaryKeys).Find(results, conditions...)

	for i := 0; i < resultValues.Len(); i++ {
		result := resultValues.Index(i)
		if scope.IndirectValue().Kind() == reflect.Slice {
			value := getRealValue(result, associationPrimaryKey)
			objects := scope.IndirectValue()
			for j := 0; j < objects.Len(); j++ {
				object := reflect.Indirect(objects.Index(j))
				if equalAsString(getRealValue(object, relation.ForeignFieldName), value) {
					object.FieldByName(field.Name).Set(result)
				}
			}
		} else {
			scope.SetColumn(field, result)
		}
	}
}

func (scope *Scope) getColumnAsArray(column string) (columns []interface{}) {
	values := scope.IndirectValue()
	switch values.Kind() {
	case reflect.Slice:
		for i := 0; i < values.Len(); i++ {
			columns = append(columns, reflect.Indirect(values.Index(i)).FieldByName(column).Interface())
		}
	case reflect.Struct:
		return []interface{}{values.FieldByName(column).Interface()}
	}
	return
}

func (scope *Scope) getColumnsAsScope(column string) *Scope {
	values := scope.IndirectValue()
	switch values.Kind() {
	case reflect.Slice:
		model := values.Type().Elem()
		if model.Kind() == reflect.Ptr {
			model = model.Elem()
		}
		fieldType, _ := model.FieldByName(column)
		var columns reflect.Value
		if fieldType.Type.Kind() == reflect.Slice {
			columns = reflect.New(reflect.SliceOf(reflect.PtrTo(fieldType.Type.Elem()))).Elem()
		} else {
			columns = reflect.New(reflect.SliceOf(reflect.PtrTo(fieldType.Type))).Elem()
		}
		for i := 0; i < values.Len(); i++ {
			column := reflect.Indirect(values.Index(i)).FieldByName(column)
			if column.Kind() == reflect.Slice {
				for i := 0; i < column.Len(); i++ {
					columns = reflect.Append(columns, column.Index(i).Addr())
				}
			} else {
				columns = reflect.Append(columns, column.Addr())
			}
		}
		return scope.New(columns.Interface())
	case reflect.Struct:
		return scope.New(values.FieldByName(column).Addr().Interface())
	}
	return nil
}
