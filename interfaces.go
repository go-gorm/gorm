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
	Explain(sql string, vars ...interface{}) string
}

// CommonDB common db interface
type CommonDB interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

type TxBeginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type TxCommiter interface {
	Commit() error
	Rollback() error
}

type BeforeCreateInterface interface {
	BeforeCreate(*DB)
}

type AfterCreateInterface interface {
	AfterCreate(*DB)
}

type BeforeUpdateInterface interface {
	BeforeUpdate(*DB)
}

type AfterUpdateInterface interface {
	AfterUpdate(*DB)
}

type BeforeSaveInterface interface {
	BeforeSave(*DB)
}

type AfterSaveInterface interface {
	AfterSave(*DB)
}

type BeforeDeleteInterface interface {
	BeforeDelete(*DB)
}

type AfterDeleteInterface interface {
	AfterDelete(*DB)
}

type AfterFindInterface interface {
	AfterFind(*DB)
}
