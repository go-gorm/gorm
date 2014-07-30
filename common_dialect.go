package gorm

import (
	"fmt"
	"reflect"
	"strings"
)

type commonDialect struct{}

func (s *commonDialect) BinVar(i int) string {
	return "?"
}

func (s *commonDialect) SupportLastInsertId() bool {
	return true
}

func (d *commonDialect) SqlTag(value reflect.Value, size int) string {
	switch value.Kind() {
	case reflect.Bool:
		return "BOOLEAN"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		return "INTEGER"
	case reflect.Int64, reflect.Uint64:
		return "BIGINT"
	case reflect.Float32, reflect.Float64:
		return "FLOAT"
	case reflect.String:
		if size > 0 && size < 65532 {
			return fmt.Sprintf("VARCHAR(%d)", size)
		} else {
			return "VARCHAR(65532)"
		}
	case reflect.Struct:
		if value.Type() == timeType {
			return "TIMESTAMP"
		}
	default:
		if _, ok := value.Interface().([]byte); ok {
			if size > 0 && size < 65532 {
				return fmt.Sprintf("BINARY(%d)", size)
			} else {
				return "BINARY(65532)"
			}
		}
	}
	panic(fmt.Sprintf("invalid sql type %s (%s) for commonDialect", value.Type().Name(), value.Kind().String()))
}

func (s *commonDialect) PrimaryKeyTag(value reflect.Value, size int) string {
	suffix_str := " NOT NULL PRIMARY KEY"
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		return "INTEGER" + suffix_str
	case reflect.Int64, reflect.Uint64:
		return "BIGINT" + suffix_str
	default:
		panic("Invalid primary key type")
	}
}

func (s *commonDialect) ReturningStr(key string) string {
	return ""
}

func (s *commonDialect) SelectFromDummyTable() string {
	return ""
}

func (s *commonDialect) Quote(key string) string {
	return fmt.Sprintf("`%s`", key)
}

func (s *commonDialect) databaseName(scope *Scope) string {
	from := strings.Index(scope.db.parent.source, "/") + 1
	to := strings.Index(scope.db.parent.source, "?")
	if to == -1 {
		to = len(scope.db.parent.source)
	}
	return scope.db.parent.source[from:to]
}

func (s *commonDialect) HasTable(scope *Scope, tableName string) bool {
	var count int
	newScope := scope.New(nil)
	newScope.Raw(fmt.Sprintf("SELECT count(*) FROM INFORMATION_SCHEMA.tables where table_name = %v AND table_schema = %v",
		newScope.AddToVars(tableName),
		newScope.AddToVars(s.databaseName(scope))))
	newScope.DB().QueryRow(newScope.Sql, newScope.SqlVars...).Scan(&count)
	return count > 0
}

func (s *commonDialect) HasColumn(scope *Scope, tableName string, columnName string) bool {
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

func (s *commonDialect) RemoveIndex(scope *Scope, indexName string) {
	scope.Raw(fmt.Sprintf("DROP INDEX %v ON %v", indexName, scope.QuotedTableName())).Exec()
}
