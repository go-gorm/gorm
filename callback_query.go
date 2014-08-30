package gorm

import (
	"reflect"
	"strings"
	"time"
)

func getColumnMap(destType reflect.Type) map[string]string {
	colToFieldMap := make(map[string]string)
	if destType != nil && destType.Kind() == reflect.Struct {
		for i := 0; i < destType.NumField(); i++ {
			field := destType.Field(i)
			if field.Anonymous {
				embeddedStructFields := getColumnMap(field.Type)
				for k, v := range embeddedStructFields {
					colToFieldMap[k] = v
				}
				continue
			}
			fieldName := field.Name
			dbColumnName := ToSnake(fieldName)
			settings := parseTagSetting(destType.Field(i).Tag.Get("gorm"))
			if colName, ok := settings["COLUMN"]; ok && colName != "" {
				dbColumnName = colName
			}
			colToFieldMap[dbColumnName] = fieldName
		}
	}
	return colToFieldMap
}

func Query(scope *Scope) {
	defer scope.Trace(time.Now())

	var (
		isSlice        bool
		isPtr          bool
		anyRecordFound bool
		destType       reflect.Type
	)

	var dest = scope.IndirectValue()
	if value, ok := scope.Get("gorm:query_destination"); ok {
		dest = reflect.Indirect(reflect.ValueOf(value))
	}

	if dest.Kind() == reflect.Slice {
		isSlice = true
		destType = dest.Type().Elem()
		if destType.Kind() == reflect.Ptr {
			isPtr = true
			destType = destType.Elem()
		}
	} else {
		scope.Search = scope.Search.clone().limit(1)
	}

	scope.prepareQuerySql()

	if !scope.HasError() {
		rows, err := scope.DB().Query(scope.Sql, scope.SqlVars...)

		if scope.Err(err) != nil {
			return
		}

		colToFieldMap := getColumnMap(destType)

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
				fieldName, ok := colToFieldMap[value]
				if !ok {
					fieldName = SnakeToUpperCamel(strings.ToLower(value))
				}
				field := elem.FieldByName(fieldName)
				if field.IsValid() {
					values = append(values, field.Addr().Interface())
				} else {
					var ignore interface{}
					values = append(values, &ignore)
				}
			}
			scope.Err(rows.Scan(values...))

			if isSlice {
				if isPtr {
					dest.Set(reflect.Append(dest, elem.Addr()))
				} else {
					dest.Set(reflect.Append(dest, elem))
				}
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
