package gorm

import "database/sql"

type sqlcommon interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

type sql_db interface {
	Begin() (*sql.Tx, error)
	SetMaxIdleConns(n int)
}

type sql_tx interface {
	Commit() error
	Rollback() error
}
