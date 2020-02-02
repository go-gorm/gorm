package gorm

import (
	"context"
	"database/sql"
)

// Dialector GORM database dialector
type Dialector interface {
	Initialize(*DB) error
	Migrator() Migrator
	BindVar(stmt Statement, v interface{}) string
}

// CommonDB common db interface
type CommonDB interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}
