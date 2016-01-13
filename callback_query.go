package gorm

import (
	"errors"
	"fmt"
	"reflect"
)

func Query(scope *Scope) {
	defer scope.trace(NowFunc())

	var (
		isSlice  bool
		isPtr    bool
		destType reflect.Type
	)

	if orderBy, ok := scope.Get("gorm:order_by_primary_key"); ok {
		if primaryKey := scope.PrimaryKey(); primaryKey != "" {
			scope.Search.Order(fmt.Sprintf("%v.%v %v", scope.QuotedTableName(), scope.Quote(primaryKey), orderBy))
		}
	}

	var dest = scope.IndirectValue()
	if value, ok := scope.Get("gorm:query_destination"); ok {
		dest = reflect.Indirect(reflect.ValueOf(value))
	}

	if kind := dest.Kind(); kind == reflect.Slice {
		isSlice = true
		destType = dest.Type().Elem()
		dest.Set(reflect.MakeSlice(dest.Type(), 0, 0))

		if destType.Kind() == reflect.Ptr {
			isPtr = true
			destType = destType.Elem()
		}
	} else if kind != reflect.Struct {
		scope.Err(errors.New("unsupported destination, should be slice or struct"))
		return
	}

	scope.prepareQuerySql()

	if !scope.HasError() {
		rows, err := scope.SqlDB().Query(scope.Sql, scope.SqlVars...)
		scope.db.RowsAffected = 0

		if scope.Err(err) != nil {
			return
		}
		defer rows.Close()

		columns, _ := rows.Columns()
		for rows.Next() {
			scope.db.RowsAffected++

			elem := dest
			if isSlice {
				elem = reflect.New(destType).Elem()
			}

			fields := scope.New(elem.Addr().Interface()).Fields()
			scope.scan(rows, columns, fields)

			if isSlice {
				if isPtr {
					dest.Set(reflect.Append(dest, elem.Addr()))
				} else {
					dest.Set(reflect.Append(dest, elem))
				}
			}
		}

		if scope.db.RowsAffected == 0 && !isSlice {
			scope.Err(RecordNotFound)
		}
	}
}

func AfterQuery(scope *Scope) {
	scope.CallMethodWithErrorCheck("AfterFind")
}

func init() {
	DefaultCallback.Query().Register("gorm:query", Query)
	DefaultCallback.Query().Register("gorm:after_query", AfterQuery)
	DefaultCallback.Query().Register("gorm:preload", Preload)
}
