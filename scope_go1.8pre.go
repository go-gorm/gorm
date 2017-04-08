// +build !go1.8

package gorm

import "database/sql"

// Begin start a transaction
func (scope *Scope) Begin() *Scope {
	if db, ok := scope.SQLDB().(sqlDb); ok {
		if tx, err := db.Begin(); err == nil {
			scope.db.db = interface{}(tx).(SQLCommon)
			scope.InstanceSet("gorm:started_transaction", true)
		}
	}
	return scope
}

func (scope *Scope) sqldbExec(query string, args ...interface{}) (sql.Result, error) {
	return scope.SQLDB().Exec(query, args...)
}

func (scope *Scope) sqldbPrepare(query string) (*sql.Stmt, error) {
	return scope.SQLDB().Prepare(query)
}

func (scope *Scope) sqldbQuery(query string, args ...interface{}) (*sql.Rows, error) {
	return scope.SQLDB().Query(query, args...)
}

func (scope *Scope) sqldbQueryRow(query string, args ...interface{}) *sql.Row {
	return scope.SQLDB().QueryRow(query, args...)
}
