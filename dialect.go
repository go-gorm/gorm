package gorm

import (
	"fmt"
	"reflect"
	"time"
)

var timeType = reflect.TypeOf(time.Time{})

type Dialect interface {
	BinVar(i int) string
	SupportLastInsertId() bool
	SqlTag(value reflect.Value, size int) string
	PrimaryKeyTag(value reflect.Value, size int) string
	ReturningStr(key string) string
	Quote(key string) string
	HasTable(scope *Scope, tableName string) bool
	HasColumn(scope *Scope, tableName string, columnName string) bool
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
	default:
		fmt.Printf("`%v` is not officially supported, running under compatibility mode.\n", driver)
		d = &commonDialect{}
	}
	return d
}
