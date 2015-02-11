package gorm

import (
	"errors"
	"fmt"
	"reflect"
)

func Preload(scope *Scope) {
	// Get Fields
	var fields map[string]*Field
	if scope.IndirectValue().Kind() == reflect.Slice {
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

					switch relation.Kind {
					case "has_one":
						sql := fmt.Sprintf("%v IN (?)", scope.Quote(relation.ForeignKey))
						scope.NewDB().Find(results, sql, scope.getColumnAsArray(primaryName))
					case "has_many":
						sql := fmt.Sprintf("%v IN (?)", scope.Quote(relation.ForeignKey))
						scope.NewDB().Find(results, sql, scope.getColumnAsArray(primaryName))
					case "belongs_to":
						scope.NewDB().Find(results, scope.getColumnAsArray(relation.ForeignKey))
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
