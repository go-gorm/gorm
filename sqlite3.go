package gorm

import (
	"fmt"
	"reflect"
	"time"
)

type sqlite3 struct {
	commonDialect
}

func (sqlite3) SqlTag(value reflect.Value, size int, autoIncrease bool) string {
	switch value.Kind() {
	case reflect.Bool:
		return "bool"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		return "integer"
	case reflect.Int64, reflect.Uint64:
		if autoIncrease {
			return "integer"
		}
		return "bigint"
	case reflect.Float32, reflect.Float64:
		return "real"
	case reflect.String:
		if size > 0 && size < 65532 {
			return fmt.Sprintf("varchar(%d)", size)
		}
		return "text"
	case reflect.Struct:
		if _, ok := value.Interface().(time.Time); ok {
			return "datetime"
		}
	default:
		if _, ok := value.Interface().([]byte); ok {
			return "blob"
		}
	}
	panic(fmt.Sprintf("invalid sql type %s (%s) for sqlite3", value.Type().Name(), value.Kind().String()))
}

func (s sqlite3) HasTable(scope *Scope, tableName string) bool {
	var count int
	s.RawScanInt(scope, &count, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", tableName)
	return count > 0
}

func (s sqlite3) HasColumn(scope *Scope, tableName string, columnName string) bool {
	var count int
	s.RawScanInt(scope, &count, fmt.Sprintf("SELECT count(*) FROM sqlite_master WHERE tbl_name = ? AND (sql LIKE '%%(\"%v\" %%' OR sql LIKE '%%,\"%v\" %%' OR sql LIKE '%%( %v %%' OR sql LIKE '%%, %v %%');\n", columnName, columnName, columnName, columnName), tableName)
	return count > 0
}

func (s sqlite3) HasIndex(scope *Scope, tableName string, indexName string) bool {
	var count int
	s.RawScanInt(scope, &count, fmt.Sprintf("SELECT count(*) FROM sqlite_master WHERE tbl_name = ? AND sql LIKE '%%INDEX %v ON%%'", indexName), tableName)
	return count > 0
}

func (sqlite3) RemoveIndex(scope *Scope, indexName string) {
	scope.Err(scope.NewDB().Exec(fmt.Sprintf("DROP INDEX %v", indexName)).Error)
}

func (sqlite3) CurrentDatabase(scope *Scope) (name string) {
	var (
		ifaces   = make([]interface{}, 3)
		pointers = make([]*string, 3)
		i        int
	)
	for i = 0; i < 3; i++ {
		ifaces[i] = &pointers[i]
	}
	if err := scope.NewDB().Raw("PRAGMA database_list").Row().Scan(ifaces...); scope.Err(err) != nil {
		return
	}
	if pointers[1] != nil {
		name = *pointers[1]
	}
	return
}
