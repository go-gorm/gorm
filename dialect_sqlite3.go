package gorm

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

type sqlite3 struct {
	commonDialect
}

// Get Data Type for Sqlite Dialect
func (sqlite3) DataTypeOf(field *StructField) string {
	var (
		dataValue, sqlType, size, additionalType = ParseFieldStructForDialect(field)
	)

	if sqlType == "" {
		switch dataValue.Kind() {
		case reflect.Bool:
			sqlType = "bool"
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
			if field.IsPrimaryKey {
				sqlType = "integer primary key autoincrement"
			} else {
				sqlType = "integer"
			}
		case reflect.Int64, reflect.Uint64:
			if field.IsPrimaryKey {
				sqlType = "integer primary key autoincrement"
			} else {
				sqlType = "bigint"
			}
		case reflect.Float32, reflect.Float64:
			sqlType = "real"
		case reflect.String:
			if size > 0 && size < 65532 {
				sqlType = fmt.Sprintf("varchar(%d)", size)
			} else {
				sqlType = "text"
			}
		case reflect.Struct:
			if _, ok := dataValue.Interface().(time.Time); ok {
				sqlType = "datetime"
			}
		default:
			if _, ok := dataValue.Interface().([]byte); ok {
				sqlType = "blob"
			}
		}
	}

	if sqlType == "" {
		panic(fmt.Sprintf("invalid sql type %s (%s) for sqlite3", dataValue.Type().Name(), dataValue.Kind().String()))
	}

	if strings.TrimSpace(additionalType) == "" {
		return sqlType
	}
	return fmt.Sprintf("%v %v", sqlType, additionalType)
}

func (s sqlite3) HasIndex(scope *Scope, tableName string, indexName string) bool {
	var count int
	s.RawScanInt(scope, &count, fmt.Sprintf("SELECT count(*) FROM sqlite_master WHERE tbl_name = ? AND sql LIKE '%%INDEX %v ON%%'", indexName), tableName)
	return count > 0
}

func (sqlite3) RemoveIndex(scope *Scope, indexName string) {
	scope.Err(scope.NewDB().Exec(fmt.Sprintf("DROP INDEX %v", indexName)).Error)
}

func (s sqlite3) HasTable(scope *Scope, tableName string) bool {
	var count int
	s.RawScanInt(scope, &count, "SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", tableName)
	return count > 0
}

func (s sqlite3) HasColumn(scope *Scope, tableName string, columnName string) bool {
	var count int
	s.RawScanInt(scope, &count, fmt.Sprintf("SELECT count(*) FROM sqlite_master WHERE tbl_name = ? AND (sql LIKE '%%(\"%v\" %%' OR sql LIKE '%%,\"%v\" %%' OR sql LIKE '%%, \"%v\" %%' OR sql LIKE '%%( %v %%' OR sql LIKE '%%, %v %%' OR sql LIKE '%%,%v %%');\n", columnName, columnName, columnName, columnName, columnName, columnName), tableName)
	return count > 0
}

func (sqlite3) currentDatabase(scope *Scope) (name string) {
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
