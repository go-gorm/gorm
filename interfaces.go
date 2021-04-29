package gorm

import (
	"context"
	"database/sql"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// Dialector GORM database dialector
type Dialector interface {
	Name() string
	Initialize(*DB) error
	Migrator(db *DB) Migrator
	DataTypeOf(*schema.Field) string
	DefaultValueOf(*schema.Field) clause.Expression
	BindVarTo(writer clause.Writer, stmt *Statement, v interface{})
	QuoteTo(clause.Writer, string)
	Explain(sql string, vars ...interface{}) string
}

// Plugin GORM plugin interface
type Plugin interface {
	Name() string
	Initialize(*DB) error
}

// ConnPool db conns pool interface
type ConnPool interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// SavePointerDialectorInterface save pointer interface
type SavePointerDialectorInterface interface {
	SavePoint(tx *DB, name string) error
	RollbackTo(tx *DB, name string) error
}

type TxBeginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

type ConnPoolBeginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (ConnPool, error)
}

type TxCommitter interface {
	Commit() error
	Rollback() error
}

// Valuer gorm valuer interface
type Valuer interface {
	GormValue(context.Context, *DB) clause.Expr
}

type GetDBConnector interface {
	GetDBConn() (*sql.DB, error)
}
