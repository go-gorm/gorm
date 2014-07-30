package gorm

import (
	"fmt"
	"reflect"
)

type sqlite3 struct{}

func (s *sqlite3) BinVar(i int) string {
	return "$$" // ?
}

func (s *sqlite3) SupportLastInsertId() bool {
	return true
}

func (s *sqlite3) SqlTag(value reflect.Value, size int) string {
	switch value.Kind() {
	case reflect.Bool:
		return "bool"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		return "integer"
	case reflect.Int64, reflect.Uint64:
		return "bigint"
	case reflect.Float32, reflect.Float64:
		return "real"
	case reflect.String:
		if size > 0 && size < 65532 {
			return fmt.Sprintf("varchar(%d)", size)
		} else {
			return "text"
		}
	case reflect.Struct:
		if value.Type() == timeType {
			return "datetime"
		}
	default:
		if _, ok := value.Interface().([]byte); ok {
			return "blob"
		}
	}
	panic(fmt.Sprintf("invalid sql type %s (%s) for sqlite3", value.Type().Name(), value.Kind().String()))
}

func (s *sqlite3) PrimaryKeyTag(value reflect.Value, size int) string {
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr, reflect.Int64, reflect.Uint64:
		return "INTEGER PRIMARY KEY"
	default:
		panic("Invalid primary key type")
	}
}

func (s *sqlite3) ReturningStr(key string) string {
	return ""
}

func (s *sqlite3) SelectFromDummyTable() string {
	return ""
}

func (s *sqlite3) Quote(key string) string {
	return fmt.Sprintf("\"%s\"", key)
}

func (s *sqlite3) HasTable(scope *Scope, tableName string) bool {
	var count int
	scope.DB().QueryRow(fmt.Sprintf("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='%v';", tableName)).Scan(&count)
	return count > 0
}

func (s *sqlite3) HasColumn(scope *Scope, tableName string, columnName string) bool {
	var count int
	scope.DB().QueryRow(fmt.Sprintf("SELECT count(*) FROM sqlite_master WHERE tbl_name = '%v' AND (sql LIKE '%%(\"%v\" %%' OR sql LIKE '%%,\"%v\" %%' OR sql LIKE '%%( %v %%' OR sql LIKE '%%, %v %%');\n", tableName, columnName, columnName, columnName, columnName)).Scan(&count)
	return count > 0
}

func (s *sqlite3) RemoveIndex(scope *Scope, indexName string) {
	scope.Raw(fmt.Sprintf("DROP INDEX %v", indexName)).Exec()
}
