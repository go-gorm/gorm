// +build go1.8

package gorm

import "database/sql"

// BeginTx start a transaction with the given options
func (scope *Scope) BeginTx(opts *sql.TxOptions) *Scope {
	if db, ok := scope.SQLDB().(sqlDb); ok {
		if tx, err := db.BeginTx(scope.DB().contextOrBackground(), opts); err == nil {
			scope.db.db = interface{}(tx).(SQLCommon)
			scope.InstanceSet("gorm:started_transaction", true)
		}
	}
	return scope
}

// Begin start a transaction
func (scope *Scope) Begin() *Scope {
	return scope.BeginTx(nil)
}

func (scope *Scope) sqldbExec(query string, args ...interface{}) (sql.Result, error) {
	return scope.SQLDB().ExecContext(scope.db.contextOrBackground(), query, args...)
}

func (scope *Scope) sqldbPrepare(query string) (*sql.Stmt, error) {
	return scope.SQLDB().PrepareContext(scope.db.contextOrBackground(), query)
}

func (scope *Scope) sqldbQuery(query string, args ...interface{}) (*sql.Rows, error) {
	return scope.SQLDB().QueryContext(scope.db.contextOrBackground(), query, args...)
}

func (scope *Scope) sqldbQueryRow(query string, args ...interface{}) *sql.Row {
	return scope.SQLDB().QueryRowContext(scope.db.contextOrBackground(), query, args...)
}
