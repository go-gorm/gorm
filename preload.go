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
	preloadMap := map[string]bool{}
	if scope.Search.preload != nil {
		fields := scope.Fields()
		isSlice := scope.IndirectValue().Kind() == reflect.Slice

		for _, preload := range scope.Search.preload {
			schema, conditions := preload.schema, preload.conditions
			keys := strings.Split(schema, ".")
			currentScope := scope
			currentFields := fields
			currentIsSlice := isSlice
			originalConditions := conditions
			conditions = []interface{}{}
			for i, key := range keys {
				// log.Printf("--> %+v\n", key)
				if !preloadMap[strings.Join(keys[:i+1], ".")] {
					if i == len(keys)-1 {
						// log.Printf("--> %+v\n", originalConditions)
						conditions = originalConditions
					}

					var found bool
					for _, field := range currentFields {
						if field.Name == key && field.Relationship != nil {
							found = true
							// log.Printf("--> %+v\n", field.Name)
							results := makeSlice(field.Struct.Type)
							relation := field.Relationship
							primaryName := currentScope.PrimaryField().Name
							associationPrimaryKey := currentScope.New(results).PrimaryField().Name

							switch relation.Kind {
							case "has_one":
								if primaryKeys := currentScope.getColumnAsArray(primaryName); len(primaryKeys) > 0 {
									condition := fmt.Sprintf("%v IN (?)", currentScope.Quote(relation.ForeignDBName))
									currentScope.NewDB().Where(condition, primaryKeys).Find(results, conditions...)

									resultValues := reflect.Indirect(reflect.ValueOf(results))
									for i := 0; i < resultValues.Len(); i++ {
										result := resultValues.Index(i)
										if currentIsSlice {
											value := getRealValue(result, relation.ForeignFieldName)
											objects := currentScope.IndirectValue()
											for j := 0; j < objects.Len(); j++ {
												if equalAsString(getRealValue(objects.Index(j), primaryName), value) {
													reflect.Indirect(objects.Index(j)).FieldByName(field.Name).Set(result)
													break
												}
											}
										} else {
											// log.Printf("--> %+v\n", result.Interface())
											err := currentScope.SetColumn(field, result)
											if err != nil {
												scope.Err(err)
												return
											}
											// printutils.PrettyPrint(currentScope.Value)
										}
									}
									// printutils.PrettyPrint(currentScope.Value)
								}
							case "has_many":
								// log.Printf("--> %+v\n", key)
								if primaryKeys := currentScope.getColumnAsArray(primaryName); len(primaryKeys) > 0 {
									condition := fmt.Sprintf("%v IN (?)", currentScope.Quote(relation.ForeignDBName))
									currentScope.NewDB().Where(condition, primaryKeys).Find(results, conditions...)
									resultValues := reflect.Indirect(reflect.ValueOf(results))
									if currentIsSlice {
										for i := 0; i < resultValues.Len(); i++ {
											result := resultValues.Index(i)
											value := getRealValue(result, relation.ForeignFieldName)
											objects := currentScope.IndirectValue()
											for j := 0; j < objects.Len(); j++ {
												object := reflect.Indirect(objects.Index(j))
												if equalAsString(getRealValue(object, primaryName), value) {
													f := object.FieldByName(field.Name)
													f.Set(reflect.Append(f, result))
													break
												}
											}
										}
										// printutils.PrettyPrint(currentScope.IndirectValue().Interface())
									} else {
										currentScope.SetColumn(field, resultValues)
									}
								}
							case "belongs_to":
								if primaryKeys := currentScope.getColumnAsArray(relation.ForeignFieldName); len(primaryKeys) > 0 {
									currentScope.NewDB().Where(primaryKeys).Find(results, conditions...)
									resultValues := reflect.Indirect(reflect.ValueOf(results))
									for i := 0; i < resultValues.Len(); i++ {
										result := resultValues.Index(i)
										if currentIsSlice {
											value := getRealValue(result, associationPrimaryKey)
											objects := currentScope.IndirectValue()
											for j := 0; j < objects.Len(); j++ {
												object := reflect.Indirect(objects.Index(j))
												if equalAsString(getRealValue(object, relation.ForeignFieldName), value) {
													object.FieldByName(field.Name).Set(result)
												}
											}
										} else {
											currentScope.SetColumn(field, result)
										}
									}
								}
							case "many_to_many":
								// currentScope.Err(errors.New("not supported relation"))
								fallthrough
							default:
								currentScope.Err(errors.New("not supported relation"))
							}
							break
						}
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
				}

				if i < len(keys)-1 {
					// TODO: update current scope
					currentScope = currentScope.getColumnsAsScope(key)
					currentFields = currentScope.Fields()
					currentIsSlice = currentScope.IndirectValue().Kind() == reflect.Slice
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
	// log.Println(values.Type(), column)
	switch values.Kind() {
	case reflect.Slice:
		fieldType, _ := values.Type().Elem().FieldByName(column)
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
