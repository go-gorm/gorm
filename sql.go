package gorm

import "database/sql"

type sqlCommon interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

type sqlDb interface {
	Begin() (*sql.Tx, error)
	SetMaxIdleConns(n int)
}

type sqlTx interface {
	Commit() error
	Rollback() error
}
