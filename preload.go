package gorm

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
)

func getFieldValue(value reflect.Value, field string) interface{} {
	result := value.FieldByName(field).Interface()
	if r, ok := result.(driver.Valuer); ok {
		result, _ = r.Value()
	}
	return result
}

func equalAsString(a interface{}, b interface{}) bool {
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func Preload(scope *Scope) {
	// Get Fields
	var fields map[string]*Field
	var isSlice bool
	if scope.IndirectValue().Kind() == reflect.Slice {
		isSlice = true
		elem := reflect.New(scope.IndirectValue().Type().Elem()).Elem()
		fields = scope.New(elem.Addr().Interface()).Fields()
	} else {
		fields = scope.Fields()
	}

	if scope.Search.Preload != nil {
		for key := range scope.Search.Preload {
			for _, field := range fields {
				if field.Name == key && field.Relationship != nil {
					results := makeSlice(field.Field)
					relation := field.Relationship
					primaryName := scope.PrimaryKeyField().Name
					associationPrimaryKey := scope.New(results).PrimaryKeyField().Name

					switch relation.Kind {
					case "has_one":
						condition := fmt.Sprintf("%v IN (?)", scope.Quote(relation.ForeignDBName()))
						scope.NewDB().Find(results, condition, scope.getColumnAsArray(primaryName))

						resultValues := reflect.Indirect(reflect.ValueOf(results))
						for i := 0; i < resultValues.Len(); i++ {
							result := resultValues.Index(i)
							if isSlice {
								value := getFieldValue(result, relation.ForeignKey)
								objects := scope.IndirectValue()
								for j := 0; j < objects.Len(); j++ {
									if equalAsString(getFieldValue(objects.Index(j), primaryName), value) {
										objects.Index(j).FieldByName(field.Name).Set(result)
										break
									}
								}
							} else {
								scope.SetColumn(field, result)
							}
						}
					case "has_many":
						condition := fmt.Sprintf("%v IN (?)", scope.Quote(relation.ForeignDBName()))
						scope.NewDB().Find(results, condition, scope.getColumnAsArray(primaryName))
						resultValues := reflect.Indirect(reflect.ValueOf(results))
						if isSlice {
							for i := 0; i < resultValues.Len(); i++ {
								result := resultValues.Index(i)
								value := getFieldValue(result, relation.ForeignKey)
								objects := scope.IndirectValue()
								for j := 0; j < objects.Len(); j++ {
									object := objects.Index(j)
									if equalAsString(getFieldValue(object, primaryName), value) {
										f := object.FieldByName(field.Name)
										f.Set(reflect.Append(f, result))
										break
									}
								}
							}
						} else {
							scope.SetColumn(field, resultValues)
						}
					case "belongs_to":
						scope.NewDB().Find(results, scope.getColumnAsArray(relation.ForeignKey))
						resultValues := reflect.Indirect(reflect.ValueOf(results))
						for i := 0; i < resultValues.Len(); i++ {
							result := resultValues.Index(i)
							if isSlice {
								value := getFieldValue(result, associationPrimaryKey)
								objects := scope.IndirectValue()
								for j := 0; j < objects.Len(); j++ {
									object := objects.Index(j)
									if equalAsString(getFieldValue(object, relation.ForeignKey), value) {
										object.FieldByName(field.Name).Set(result)
										break
									}
								}
							} else {
								scope.SetColumn(field, result)
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

func makeSlice(value reflect.Value) interface{} {
	typ := value.Type()
	if value.Kind() == reflect.Slice {
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
		for i := 0; i < values.Len(); i++ {
			value := values.Index(i)
			primaryKeys = append(primaryKeys, value.FieldByName(column).Interface())
		}
	case reflect.Struct:
		return []interface{}{values.FieldByName(column).Interface()}
	}
	return
}
