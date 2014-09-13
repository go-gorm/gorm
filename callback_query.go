package gorm

import (
	"fmt"
	"reflect"
)

func Query(scope *Scope) {
	defer scope.Trace(NowFunc())

	var (
		isSlice        bool
		isPtr          bool
		anyRecordFound bool
		destType       reflect.Type
	)

	var dest = scope.IndirectValue()
	if value, ok := scope.InstanceGet("gorm:query_destination"); ok {
		dest = reflect.Indirect(reflect.ValueOf(value))
	}

	if orderBy, ok := scope.InstanceGet("gorm:order_by_primary_key"); ok {
		if primaryKey := scope.PrimaryKey(); primaryKey != "" {
			scope.Search = scope.Search.clone().order(fmt.Sprintf("%v.%v %v", scope.QuotedTableName(), primaryKey, orderBy))
		}
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

		defer rows.Close()
		for rows.Next() {
			anyRecordFound = true
			elem := dest
			if isSlice {
				elem = reflect.New(destType).Elem()
			}

			columns, _ := rows.Columns()
			var values []interface{}
			fields := scope.New(elem.Addr().Interface()).Fields()
			for _, value := range columns {
				if field, ok := fields[value]; ok {
					values = append(values, field.Field.Addr().Interface())
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
