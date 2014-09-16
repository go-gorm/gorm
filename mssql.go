package gorm

import (
	"fmt"
	"reflect"
	"strings"
)

type mssql struct{}

func (s *mssql) BinVar(i int) string {
	return "$$" // ?
}

func (s *mssql) SupportLastInsertId() bool {
	return true
}

func (s *mssql) HasTop() bool {
	return true
}

func (d *mssql) SqlTag(value reflect.Value, size int) string {
	switch value.Kind() {
	case reflect.Bool:
		return "bit"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		return "int"
	case reflect.Int64, reflect.Uint64:
		return "bigint"
	case reflect.Float32, reflect.Float64:
		return "float"
	case reflect.String:
		if size > 0 && size < 65532 {
			return fmt.Sprintf("nvarchar(%d)", size)
		} else {
			return "text"
		}
	case reflect.Struct:
		if value.Type() == timeType {
			return "datetime2"
		}
	default:
		if _, ok := value.Interface().([]byte); ok {
			if size > 0 && size < 65532 {
				return fmt.Sprintf("varchar(%d)", size)
			} else {
				return "text"
			}
		}
	}
	panic(fmt.Sprintf("invalid sql type %s (%s) for mssql", value.Type().Name(), value.Kind().String()))
}

func (s *mssql) PrimaryKeyTag(value reflect.Value, size int) string {
	suffix_str := " IDENTITY(1,1) PRIMARY KEY"
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uintptr:
		return "int" + suffix_str
	case reflect.Int64, reflect.Uint64:
		return "bigint" + suffix_str
	default:
		panic("Invalid primary key type")
	}
}

func (s *mssql) ReturningStr(key string) string {
	return ""
}

func (s *mssql) SelectFromDummyTable() string {
	return ""
}

func (s *mssql) Quote(key string) string {
	return fmt.Sprintf(" \"%s\"", key)
}

func (s *mssql) databaseName(scope *Scope) string {
	dbStr := strings.Split(scope.db.parent.source, ";")
	for _, value := range dbStr {
		s := strings.Split(value, "=")
		if s[0] == "database" {
			return s[1]
		}
	}
	return ""
}

func (s *mssql) HasTable(scope *Scope, tableName string) bool {
	var count int
	newScope := scope.New(nil)
	newScope.Raw(fmt.Sprintf("SELECT count(*) FROM INFORMATION_SCHEMA.tables where table_name = %v AND table_catalog = %v",
		newScope.AddToVars(tableName),
		newScope.AddToVars(s.databaseName(scope))))
	newScope.DB().QueryRow(newScope.Sql, newScope.SqlVars...).Scan(&count)
	return count > 0
}

func (s *mssql) HasColumn(scope *Scope, tableName string, columnName string) bool {
	var count int
	newScope := scope.New(nil)
	newScope.Raw(fmt.Sprintf("SELECT count(*) FROM information_schema.columns WHERE TABLE_CATALOG = %v AND table_name = %v AND column_name = %v",
		newScope.AddToVars(s.databaseName(scope)),
		newScope.AddToVars(tableName),
		newScope.AddToVars(columnName),
	))
	newScope.DB().QueryRow(newScope.Sql, newScope.SqlVars...).Scan(&count)
	return count > 0
}

func (s *mssql) RemoveIndex(scope *Scope, indexName string) {
	scope.Raw(fmt.Sprintf("DROP INDEX %v ON %v", indexName, scope.QuotedTableName())).Exec()
}
