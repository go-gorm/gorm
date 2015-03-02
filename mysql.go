package gorm

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

type mysql struct{}

func (s *mysql) BinVar(i int) string {
	return "$$" // ?
}

func (s *mysql) SupportLastInsertId() bool {
	return true
}

func (s *mysql) HasTop() bool {
	return false
}

func (s *mysql) SqlTag(value reflect.Value, size int) string {
	switch value.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		return "int"
	case reflect.Int64, reflect.Uint64:
		return "bigint"
	case reflect.Float32, reflect.Float64:
		return "double"
	case reflect.String:
		if size > 0 && size < 65532 {
			return fmt.Sprintf("varchar(%d)", size)
		}
		return "longtext"
	case reflect.Struct:
		if _, ok := value.Interface().(time.Time); ok {
			return "timestamp NULL"
		}
	default:
		if _, ok := value.Interface().([]byte); ok {
			if size > 0 && size < 65532 {
				return fmt.Sprintf("varbinary(%d)", size)
			}
			return "longblob"
		}
	}
	panic(fmt.Sprintf("invalid sql type %s (%s) for mysql", value.Type().Name(), value.Kind().String()))
}

func (s *mysql) PrimaryKeyTag(value reflect.Value, size int) string {
	suffix := " NOT NULL AUTO_INCREMENT PRIMARY KEY"
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		return "int" + suffix
	case reflect.Int64, reflect.Uint64:
		return "bigint" + suffix
	default:
		panic("Invalid primary key type")
	}
}

func (s *mysql) ReturningStr(tableName, key string) string {
	return ""
}

func (s *mysql) SelectFromDummyTable() string {
	return "FROM DUAL"
}

func (s *mysql) Quote(key string) string {
	return fmt.Sprintf("`%s`", key)
}

func (s *mysql) databaseName(scope *Scope) string {
	from := strings.Index(scope.db.parent.source, "/") + 1
	to := strings.Index(scope.db.parent.source, "?")
	if to == -1 {
		to = len(scope.db.parent.source)
	}
	return scope.db.parent.source[from:to]
}

func (s *mysql) HasTable(scope *Scope, tableName string) bool {
	var count int
	scope.NewDB().Raw("SELECT count(*) FROM INFORMATION_SCHEMA.tables where table_name = ? AND table_schema = ?", tableName, s.databaseName(scope)).Row().Scan(&count)
	return count > 0
}

func (s *mysql) HasColumn(scope *Scope, tableName string, columnName string) bool {
	var count int
	scope.NewDB().Raw("SELECT count(*) FROM information_schema.columns WHERE table_schema = ? AND table_name = ? AND column_name = ?", s.databaseName(scope), tableName, columnName).Row().Scan(&count)
	return count > 0
}

func (s *mysql) HasIndex(scope *Scope, tableName string, indexName string) bool {
	var count int
	scope.NewDB().Raw("SELECT count(*) FROM INFORMATION_SCHEMA.STATISTICS where table_name = ? AND index_name = ?", tableName, indexName).Row().Scan(&count)
	return count > 0
}

func (s *mysql) RemoveIndex(scope *Scope, indexName string) {
	scope.NewDB().Exec(fmt.Sprintf("DROP INDEX %v ON %v", indexName, scope.QuotedTableName()))
}
