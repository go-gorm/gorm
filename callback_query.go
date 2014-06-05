package gorm

import (
	"reflect"
	"time"
)

func Query(scope *Scope) {
	defer scope.Trace(time.Now())

	var (
		isSlice        bool
		anyRecordFound bool
		destType       reflect.Type
	)

	var dest = reflect.Indirect(reflect.ValueOf(scope.Value))
	if value, ok := scope.Get("gorm:query_destination"); ok {
		dest = reflect.Indirect(reflect.ValueOf(value))
	}

	if dest.Kind() == reflect.Slice {
		isSlice = true
		destType = dest.Type().Elem()
	} else {
		scope.Search = scope.Search.clone().limit(1)
	}

	scope.prepareQuerySql()

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

		if !anyRecordFound {
			scope.Err(RecordNotFound)
		}
	}
}

func AfterQuery(scope *Scope) {
	scope.CallMethod("AfterFind")
}

func init() {
	DefaultCallback.Query().Register("gorm:query", Query)
	DefaultCallback.Query().Register("gorm:after_query", AfterQuery)
}
