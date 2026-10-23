package gorm

import (
	"context"
	"database/sql"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// Dialector adapts schema mapping, SQL generation, connection initialization,
// and migration to a database engine. Pass a driver implementation to Open.
// Optional capabilities such as error translation and savepoints are separate
// interfaces. See https://gorm.io/docs/write_driver.html.
type Dialector interface {
	// Name returns the dialect identifier used in configuration and diagnostics.
	Name() string
	// Initialize installs the connection pool, callbacks, and dialect-specific builders on
	// db. Return a setup error so Open can report failure.
	Initialize(*DB) error
	// Migrator returns the dialect-specific schema manager bound to db and its current
	// session.
	Migrator(db *DB) Migrator
	// DataTypeOf maps parsed field metadata to a database column type; naming and sizing
	// depend on the dialect.
	DataTypeOf(*schema.Field) string
	// DefaultValueOf returns the expression used when an INSERT needs the database default
	// for a field.
	DefaultValueOf(*schema.Field) clause.Expression
	// BindVarTo writes the next dialect-specific value placeholder. Statement.Vars already
	// contains the value being bound.
	BindVarTo(writer clause.Writer, stmt *Statement, v interface{})
	// QuoteTo writes a quoted SQL identifier, including dialect-specific handling of
	// qualified names. It does not bind a data value.
	QuoteTo(clause.Writer, string)
	// Explain formats SQL and values for human-readable diagnostics. Its output is not
	// guaranteed safe for execution.
	Explain(sql string, vars ...interface{}) string
}

// Plugin extends a DB during initialization, usually by registering callbacks.
// Name identifies the plugin in the shared registry; Initialize must return an
// error if setup fails. Register plugins with DB.Use before concurrent use.
type Plugin interface {
	// Name returns the stable unique registration key used by DB.Use.
	Name() string
	// Initialize installs the plugin on db. It runs before the plugin is recorded in the
	// registry and must report setup errors.
	Initialize(*DB) error
}

// ParamsFilter lets a logger transform SQL text and parameters for tracing.
// It only changes the logged representation, not the statement sent to the database.
// Implement it to redact sensitive values before logs are formatted.
type ParamsFilter interface {
	// ParamsFilter returns the SQL and arguments to include in a trace. Changes do not
	// affect database execution.
	ParamsFilter(ctx context.Context, sql string, params ...interface{}) (string, []interface{})
}

// ConnPool is the execution surface used by statements. *sql.DB, *sql.Tx,
// and *sql.Conn satisfy it; the name does not require a multi-connection pool.
// Transaction begin/commit, ping, and underlying-DB access are optional interfaces.
// Implementations must honor context and database/sql row and statement ownership.
type ConnPool interface {
	// PrepareContext prepares query using ctx. The owner of the resulting statement must
	// eventually close it.
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	// ExecContext executes query with bound args and returns driver result metadata or an
	// error.
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	// QueryContext executes query with bound args. The caller owns the returned rows and
	// must close them and check iteration errors.
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	// QueryRowContext executes a single-row query; database/sql defers query and no-row
	// errors until the returned row is scanned.
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// SavePointerDialectorInterface is the optional dialector capability used by
// SavePoint, RollbackTo, and nested Transaction. Implementations execute on the
// supplied transaction and are responsible for dialect-specific savepoint syntax.
type SavePointerDialectorInterface interface {
	// SavePoint creates name within tx. The name is an identifier, not a bound data
	// parameter.
	SavePoint(tx *DB, name string) error
	// RollbackTo undoes changes after name while preserving the enclosing transaction.
	RollbackTo(tx *DB, name string) error
}

// TxBeginner starts a database/sql transaction with the supplied context and
// options. DB.Begin uses it when the connection returns a concrete *sql.Tx.
type TxBeginner interface {
	// BeginTx starts a transaction governed by ctx and opts; nil opts requests the driver
	// defaults.
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// ConnPoolBeginner starts a transaction returning a ConnPool wrapper rather
// than a concrete *sql.Tx. The returned pool should also implement TxCommitter so
// GORM can finish the transaction; prepared-statement wrappers use this capability.
type ConnPoolBeginner interface {
	// BeginTx starts a transaction wrapped as ConnPool. Return an error without a usable
	// transaction when beginning fails.
	BeginTx(ctx context.Context, opts *sql.TxOptions) (ConnPool, error)
}

// TxCommitter provides transaction completion for DB.Commit and DB.Rollback.
// A transaction is single-use; completion errors must be returned to the caller.
type TxCommitter interface {
	// Commit attempts to persist the transaction. A non-nil error means the caller must not
	// report successful completion.
	Commit() error
	// Rollback aborts the transaction; return driver errors rather than silently reporting
	// success.
	Rollback() error
}

// Tx combines statement execution, transaction completion, and binding a
// prepared *sql.Stmt to the transaction with StmtContext. PreparedStmtTX relies on
// this capability to execute cached statements on the correct transaction.
type Tx interface {
	ConnPool
	TxCommitter
	// StmtContext returns a transaction-bound form of stmt so execution uses this
	// transaction rather than its originating pool.
	StmtContext(ctx context.Context, stmt *sql.Stmt) *sql.Stmt
}

// Valuer converts a Go value into a SQL expression using the operation context
// and DB. It supports context-dependent values and database expressions for custom
// types. Return trusted SQL with values in clause.Expr.Vars, not interpolated data.
// See https://gorm.io/docs/data_types.html#GormValuerInterface.
type Valuer interface {
	// GormValue returns an expression for the value in the current operation. Put dynamic
	// data in Vars and keep the SQL template trusted.
	GormValue(context.Context, *DB) clause.Expr
}

// GetDBConnector exposes the underlying *sql.DB through a connection wrapper.
// DB.DB uses it for pool access; returning a pool does not make direct operations
// on that pool participate in a transaction held by the wrapper.
type GetDBConnector interface {
	// GetDBConn returns the underlying shared pool, or an error when the wrapper cannot
	// expose one.
	GetDBConn() (*sql.DB, error)
}

// Rows is the row-iteration surface consumed by Scan. *sql.Rows satisfies it.
// The owner must position rows with Next as required by ScanMode, check Err, and
// close them; Scan itself does not take ownership of the connection or close rows.
type Rows interface {
	// Columns returns result-column names in scan order.
	Columns() ([]string, error)
	// ColumnTypes returns driver-provided metadata used to allocate scan destinations when
	// a schema is unavailable.
	ColumnTypes() ([]*sql.ColumnType, error)
	// Next positions the cursor on the next row; false means exhaustion or failure,
	// distinguished by Err.
	Next() bool
	// Scan copies the current row into writable destination pointers in column order.
	Scan(dest ...interface{}) error
	// Err reports an error encountered during row iteration, including after Next returns
	// false.
	Err() error
	// Close releases resources held by the result set. The row owner must call it even when
	// iteration stops early.
	Close() error
}

// ErrorTranslator is an optional dialector capability mapping native errors
// to GORM sentinels. DB.AddError invokes it only when Config.TranslateError is true.
// Return unrecognized errors unchanged so driver-specific information is preserved.
type ErrorTranslator interface {
	// Translate maps a recognized native error to a portable error and preserves errors it
	// cannot classify.
	Translate(err error) error
}
