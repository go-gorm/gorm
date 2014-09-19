package gorm

import (
	"fmt"
	"strings"
	"reflect"
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

func (d *mysql) SqlTag(value reflect.Value, size int) string {
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
		} else {
			return "longtext"
		}
	case reflect.Struct:
		if value.Type() == timeType {
			return "datetime"
		}
	default:
		if _, ok := value.Interface().([]byte); ok {
			if size > 0 && size < 65532 {
				return fmt.Sprintf("varbinary(%d)", size)
			} else {
				return "longblob"
			}
		}
	}
	panic(fmt.Sprintf("invalid sql type %s (%s) for mysql", value.Type().Name(), value.Kind().String()))
}

func (s *mysql) PrimaryKeyTag(value reflect.Value, size int) string {
	suffix_str := " NOT NULL AUTO_INCREMENT PRIMARY KEY"
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		return "int" + suffix_str
	case reflect.Int64, reflect.Uint64:
		return "bigint" + suffix_str
	default:
		panic("Invalid primary key type")
	}
}

func (s *mysql) ReturningStr(key string) string {
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
	newScope := scope.New(nil)
	newScope.Raw(fmt.Sprintf("SELECT count(*) FROM INFORMATION_SCHEMA.tables where table_name = %v AND table_schema = %v",
		newScope.AddToVars(tableName),
		newScope.AddToVars(s.databaseName(scope))))
	newScope.DB().QueryRow(newScope.Sql, newScope.SqlVars...).Scan(&count)
	return count > 0
}

func (s *mysql) HasColumn(scope *Scope, tableName string, columnName string) bool {
	var count int
	newScope := scope.New(nil)
	newScope.Raw(fmt.Sprintf("SELECT count(*) FROM information_schema.columns WHERE table_schema = %v AND table_name = %v AND column_name = %v",
		newScope.AddToVars(s.databaseName(scope)),
		newScope.AddToVars(tableName),
		newScope.AddToVars(columnName),
	))
	newScope.DB().QueryRow(newScope.Sql, newScope.SqlVars...).Scan(&count)
	return count > 0
}

func (s *mysql) RemoveIndex(scope *Scope, indexName string) {
	scope.Raw(fmt.Sprintf("DROP INDEX %v ON %v", indexName, scope.QuotedTableName())).Exec()
}
