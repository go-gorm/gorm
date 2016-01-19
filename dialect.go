package gorm

import (
	"fmt"
	"reflect"
)

// Dialect interface contains behaviors that differ across SQL database
type Dialect interface {
	// BindVar return the placeholder for actual values in SQL statements, in many dbs it is "?", Postgres using $1
	BindVar(i int) string
	// Quote quotes field name to avoid SQL parsing exceptions by using a reserved word as a field name
	Quote(key string) string
	// DataTypeOf return data's sql type
	DataTypeOf(value reflect.Value, size int, autoIncrease bool) string

	// HasIndex check has index or not
	HasIndex(scope *Scope, tableName string, indexName string) bool
	// RemoveIndex remove index
	RemoveIndex(scope *Scope, indexName string)
	// HasTable check has table or not
	HasTable(scope *Scope, tableName string) bool
	// HasColumn check has column or not
	HasColumn(scope *Scope, tableName string, columnName string) bool

	// LimitAndOffsetSQL return generate SQL with limit and offset, as mssql has special case
	LimitAndOffsetSQL(limit, offset int) string
	// SelectFromDummyTable return select values, for most dbs, `SELECT values` just works, mysql needs `SELECT value FROM DUAL`
	SelectFromDummyTable() string
	// LastInsertIdReturningSuffix most dbs support LastInsertId, but postgres needs to use `RETURNING`
	LastInsertIdReturningSuffix(tableName, columnName string) string
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
