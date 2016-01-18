package gorm

import (
	"fmt"
	"reflect"
)

type Dialect interface {
	BinVar(i int) string
	Quote(key string) string
	SqlTag(value reflect.Value, size int, autoIncrease bool) string

	HasIndex(scope *Scope, tableName string, indexName string) bool
	RemoveIndex(scope *Scope, indexName string)
	HasTable(scope *Scope, tableName string) bool
	HasColumn(scope *Scope, tableName string, columnName string) bool
	CurrentDatabase(scope *Scope) string

	ReturningStr(tableName, key string) string
	LimitAndOffsetSQL(limit, offset int) string
	SelectFromDummyTable() string
	SupportLastInsertId() bool
}

func NewDialect(driver string) Dialect {
	var d Dialect
	switch driver {
	case "postgres":
		d = &postgres{}
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
