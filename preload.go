package gorm

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
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
	if scope.Search.preload != nil {
		fields := scope.Fields()
		isSlice := scope.IndirectValue().Kind() == reflect.Slice

		for key, conditions := range scope.Search.preload {
			for _, field := range fields {
				if field.Name == key && field.Relationship != nil {
					results := makeSlice(field.Struct.Type)
					relation := field.Relationship
					primaryName := scope.PrimaryField().Name
					associationPrimaryKey := scope.New(results).PrimaryField().Name

					switch relation.Kind {
					case "has_one":
						if primaryKeys := scope.getColumnAsArray(primaryName); len(primaryKeys) > 0 {
							condition := fmt.Sprintf("%v IN (?)", scope.Quote(relation.ForeignDBName))
							scope.NewDB().Where(condition, primaryKeys).Find(results, conditions...)

							resultValues := reflect.Indirect(reflect.ValueOf(results))
							for i := 0; i < resultValues.Len(); i++ {
								result := resultValues.Index(i)
								if isSlice {
									value := getRealValue(result, relation.ForeignFieldName)
									objects := scope.IndirectValue()
									for j := 0; j < objects.Len(); j++ {
										if equalAsString(getRealValue(objects.Index(j), primaryName), value) {
											reflect.Indirect(objects.Index(j)).FieldByName(field.Name).Set(result)
											break
										}
									}
								} else {
									scope.SetColumn(field, result)
								}
							}
						}
					case "has_many":
						if primaryKeys := scope.getColumnAsArray(primaryName); len(primaryKeys) > 0 {
							condition := fmt.Sprintf("%v IN (?)", scope.Quote(relation.ForeignDBName))
							scope.NewDB().Where(condition, primaryKeys).Find(results, conditions...)
							resultValues := reflect.Indirect(reflect.ValueOf(results))
							if isSlice {
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
					case "belongs_to":
						if primaryKeys := scope.getColumnAsArray(relation.ForeignFieldName); len(primaryKeys) > 0 {
							scope.NewDB().Where(primaryKeys).Find(results, conditions...)
							resultValues := reflect.Indirect(reflect.ValueOf(results))
							for i := 0; i < resultValues.Len(); i++ {
								result := resultValues.Index(i)
								if isSlice {
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
					case "many_to_many":
						scope.Err(errors.New("not supported relation"))
					default:
						scope.Err(errors.New("not supported relation"))
					}
					break
				}
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

func (scope *Scope) getColumnAsArray(column string) (primaryKeys []interface{}) {
	values := scope.IndirectValue()
	switch values.Kind() {
	case reflect.Slice:
		primaryKeyMap := map[interface{}]bool{}
		for i := 0; i < values.Len(); i++ {
			primaryKeyMap[reflect.Indirect(values.Index(i)).FieldByName(column).Interface()] = true
		}
		for key := range primaryKeyMap {
			primaryKeys = append(primaryKeys, key)
		}
	case reflect.Struct:
		return []interface{}{values.FieldByName(column).Interface()}
	}
	return
}
