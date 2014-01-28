package gorm

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

func Query(scope *Scope) {
	defer scope.Trace(time.Now())

	inlineCondition, ok := scope.Get("gorm:inline_condition")
	if ok {
		inlineConditions := inlineCondition.([]interface{})
		if len(inlineConditions) > 0 {
			scope.Search = scope.Search.clone().where(inlineConditions[0], inlineConditions[1:]...)
		}
	}

	var (
		isSlice        bool
		anyRecordFound bool
		destType       reflect.Type
	)

	var dest = reflect.Indirect(reflect.ValueOf(scope.Value))

	if dest.Kind() == reflect.Slice {
		isSlice = true
		destType = dest.Type().Elem()
	} else {
		scope.Search = scope.Search.clone().limit(1)
	}

	if scope.Search.raw {
		scope.Raw(strings.TrimLeft(scope.CombinedConditionSql(), "WHERE "))
	} else {
		scope.Raw(fmt.Sprintf("SELECT %v FROM %v %v", scope.SelectSql(), scope.TableName(), scope.CombinedConditionSql()))
	}

	if !scope.HasError() {
		rows, err := scope.DB().Query(scope.Sql, scope.SqlVars...)

		if scope.Err(err) != nil {
			return
		}

		defer rows.Close()
		for rows.Next() {
			anyRecordFound = true
			elem := dest
			if isSlice {
				elem = reflect.New(destType).Elem()
			}

			columns, _ := rows.Columns()
			var values []interface{}
			for _, value := range columns {
				field := elem.FieldByName(snakeToUpperCamel(value))
				if field.IsValid() {
					values = append(values, field.Addr().Interface())
				} else {
					var ignore interface{}
					values = append(values, &ignore)
				}
			}
			scope.Err(rows.Scan(values...))

			if isSlice {
				dest.Set(reflect.Append(dest, elem))
			}
		}

		if !anyRecordFound && !isSlice {
			scope.Err(RecordNotFound)
		}
	}
}

func AfterQuery(scope *Scope) {
	scope.CallMethod("AfterFind")
}

func init() {
	DefaultCallback.Query().Register("query", Query)
	DefaultCallback.Query().Register("after_query", AfterQuery)
}
