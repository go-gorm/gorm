package gorm

import (
	"fmt"
	"reflect"
)

type Dialect interface {
	BinVar(i int) string
	SupportLastInsertId() bool
	HasTop() bool
	SqlTag(value reflect.Value, size int, autoIncrease bool) string
	ReturningStr(tableName, key string) string
	SelectFromDummyTable() string
	Quote(key string) string
	HasTable(scope *Scope, tableName string) bool
	HasColumn(scope *Scope, tableName string, columnName string) bool
	HasIndex(scope *Scope, tableName string, indexName string) bool
	RemoveIndex(scope *Scope, indexName string)
	CurrentDatabase(scope *Scope) string
}

func NewDialect(driver string) Dialect {
	var d Dialect
	switch driver {
	case "postgres":
		d = &postgres{}
	case "foundation":
		d = &foundation{}
	case "mysql":
		d = &mysql{}
	case "sqlite3":
		d = &sqlite3{}
	case "mssql":
		d = &mssql{}
	default:
		fmt.Printf("`%v` is not officially supported, running under compatibility mode.\n", driver)
		d = &commonDialect{}
	}
	return d
}
