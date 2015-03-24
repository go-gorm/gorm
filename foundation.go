package gorm

import (
	"fmt"
	"reflect"
	"time"
)

type foundation struct {
	commonDialect
}

func (foundation) BinVar(i int) string {
	return fmt.Sprintf("$%v", i)
}

func (foundation) SupportLastInsertId() bool {
	return false
}

func (foundation) SqlTag(value reflect.Value, size int, autoIncrease bool) string {
	switch value.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		if autoIncrease {
			return "serial"
		}
		return "int"
	case reflect.Int64, reflect.Uint64:
		if autoIncrease {
			return "bigserial"
		}
		return "bigint"
	case reflect.Float32, reflect.Float64:
		return "double"
	case reflect.String:
		if size > 0 && size < 65532 {
			return fmt.Sprintf("varchar(%d)", size)
		}
		return "clob"
	case reflect.Struct:
		if _, ok := value.Interface().(time.Time); ok {
			return "datetime"
		}
	default:
		if _, ok := value.Interface().([]byte); ok {
			return "blob"
		}
	}
	panic(fmt.Sprintf("invalid sql type %s (%s) for foundation", value.Type().Name(), value.Kind().String()))
}

func (f foundation) ReturningStr(tableName, key string) string {
	return fmt.Sprintf("RETURNING %v.%v", f.Quote(tableName), key)
}

func (foundation) HasTable(scope *Scope, tableName string) bool {
	var count int
	scope.NewDB().Raw("SELECT count(*) FROM INFORMATION_SCHEMA.tables WHERE table_schema = current_schema AND table_type = 'TABLE' AND table_name = ?", tableName).Row().Scan(&count)
	return count > 0
}

func (foundation) HasColumn(scope *Scope, tableName string, columnName string) bool {
	var count int
	scope.NewDB().Raw("SELECT count(*) FROM INFORMATION_SCHEMA.columns WHERE table_schema = current_schema AND table_name = ? AND column_name = ?", tableName, columnName).Row().Scan(&count)
	return count > 0
}

func (f foundation) RemoveIndex(scope *Scope, indexName string) {
	scope.NewDB().Exec(fmt.Sprintf("DROP INDEX %v", f.Quote(indexName)))
}

func (foundation) HasIndex(scope *Scope, tableName string, indexName string) bool {
	var count int
	scope.NewDB().Raw("SELECT count(*) FROM INFORMATION_SCHEMA.indexes WHERE table_schema = current_schema AND table_name = ? AND index_name = ?", tableName, indexName).Row().Scan(&count)
	return count > 0
}
