package gorm

import (
	"context"
	"database/sql"

	"github.com/jinzhu/gorm/schema"
)

// Dialector GORM database dialector
type Dialector interface {
	Initialize(*DB) error
	Migrator(db *DB) Migrator
	DataTypeOf(*schema.Field) string
	BindVar(stmt *Statement, v interface{}) string
	QuoteChars() [2]byte
}

// CommonDB common db interface
type CommonDB interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}
