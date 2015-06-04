package gorm

import (
	"fmt"
	"reflect"
	"time"
)

type crate struct {
	commonDialect
}

func (crate) SupportLastInsertId() bool {
	return false
}

func (crate) SqlTag(value reflect.Value, size int, autoIncrease bool) string {
	//Crate doesn't support autoIncrease
	switch value.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int8, reflect.Int16, reflect.Uint8, reflect.Uint16:
		return "short"
	case reflect.Int, reflect.Int32, reflect.Uint, reflect.Uint32, reflect.Uintptr:
		return "integer"
	case reflect.Int64, reflect.Uint64:
		return "double"
	case reflect.Float32, reflect.Float64:
		return "float"
	case reflect.String:
		return "string"
	case reflect.Struct:
		if _, ok := value.Interface().(time.Time); ok {
			return "timestamp NULL"
		}
	default:
		if _, ok := value.Interface().([]byte); ok {
			return "object"
		}
	}
	panic(fmt.Sprintf("invalid sql type %s (%s) for mysql", value.Type().Name(), value.Kind().String()))
}

func (c crate) HasTable(scope *Scope, tableName string) bool {
	var count int
	scope.NewDB().Raw("SELECT count(*) FROM INFORMATION_SCHEMA.TABLES WHERE table_name = ? AND schema_name = 'doc'", tableName).Row().Scan(&count)
	return count > 0
}

func (c crate) HasColumn(scope *Scope, tableName string, columnName string) bool {
	var count int
	scope.NewDB().Raw("SELECT count(*) FROM INFORMATION_SCHEMA.COLUMNS WHERE schema_name = 'doc' AND table_name = ? AND column_name = ?", tableName, columnName).Row().Scan(&count)
	return count > 0
}

func (crate) Quote(key string) string {
	return fmt.Sprintf(`"%s"`, key)
}

func (crate) QueryTerminator() string {
	return ""
}
