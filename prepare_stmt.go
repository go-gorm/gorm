package gorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"reflect"
	"sync"
	"time"

	"gorm.io/gorm/internal/stmt_store"
)

// PreparedStmtDB wraps a ConnPool with an SQL-text-keyed prepared-statement cache.
// Sessions may share its Stmts and Mux. Use Config.PrepareStmt or Session.PrepareStmt
// for normal setup; Close evicts cached statements without closing the connection pool.
type PreparedStmtDB struct {
	// Stmts owns cached prepared statements. Entries are evicted on capacity, expiration,
	// explicit deletion, or a bad-connection error. Do not replace the store while
	// operations are running.
	Stmts stmt_store.Store
	// Mux coordinates lookup/creation and bulk eviction across wrappers sharing the same
	// store.
	Mux *sync.RWMutex
	ConnPool
}

// NewPreparedStmtDB wraps connPool with a fresh prepared-statement store and lock.
// maxSize bounds cached SQL templates; a non-positive value defaults to math.MaxInt. A
// non-positive ttl defaults to 24 hours. Eviction closes statements asynchronously. The
// wrapper does not own the lifetime of connPool; close its underlying pool separately
// at shutdown.
func NewPreparedStmtDB(connPool ConnPool, maxSize int, ttl time.Duration) *PreparedStmtDB {
	return &PreparedStmtDB{
		ConnPool: connPool,                     // Assigns the provided connection pool to manage database connections.
		Stmts:    stmt_store.New(maxSize, ttl), // Initializes a new statement store with the specified maximum size and TTL.
		Mux:      &sync.RWMutex{},              // Sets up a read-write mutex for synchronizing access to the statement store.
	}
}

// GetDBConn exposes the underlying shared *sql.DB directly or through GetDBConnector.
// It returns ErrInvalidDB when the wrapped connection cannot expose a pool.
func (db *PreparedStmtDB) GetDBConn() (*sql.DB, error) {
	if sqldb, ok := db.ConnPool.(*sql.DB); ok {
		return sqldb, nil
	}

	if dbConnector, ok := db.ConnPool.(GetDBConnector); ok && dbConnector != nil {
		return dbConnector.GetDBConn()
	}

	return nil, ErrInvalidDB
}

// Close removes the live entries returned by the store's Keys method while
// holding the cache lock. Expired entries can still await background cleanup.
// Eviction schedules statement closure asynchronously; Close does not wait for
// all closures, stop the expiry worker, or close the underlying pool. The store
// remains usable, and other sessions sharing it observe the eviction.
func (db *PreparedStmtDB) Close() {
	db.Mux.Lock()
	defer db.Mux.Unlock()

	for _, key := range db.Stmts.Keys() {
		db.Stmts.Delete(key)
	}
}

// Reset clears the cache by calling Close.
//
// Deprecated: use Close instead.
func (db *PreparedStmtDB) Reset() {
	db.Close()
}

// prepare looks up query, rejecting transaction-only entries for nontransactional use,
// then rechecks under the write lock before creating a missing statement. Store.New
// receives ownership of releasing that lock around preparation; changing this contract
// can deadlock concurrent users.
func (db *PreparedStmtDB) prepare(ctx context.Context, conn ConnPool, isTransaction bool, query string) (_ *stmt_store.Stmt, err error) {
	db.Mux.RLock()
	if db.Stmts != nil {
		if stmt, ok := db.Stmts.Get(query); ok && (!stmt.Transaction || isTransaction) {
			db.Mux.RUnlock()
			return stmt, stmt.Error()
		}
	}
	db.Mux.RUnlock()

	// retry
	db.Mux.Lock()
	if db.Stmts != nil {
		if stmt, ok := db.Stmts.Get(query); ok && (!stmt.Transaction || isTransaction) {
			db.Mux.Unlock()
			return stmt, stmt.Error()
		}
	}

	return db.Stmts.New(ctx, query, isTransaction, conn, db.Mux)
}

// BeginTx starts a transaction on the wrapped pool and returns a PreparedStmtTX sharing
// this cache. The pool must implement TxBeginner or ConnPoolBeginner, and a wrapped
// transaction must implement Tx. Always check the error even if a non-nil wrapper is
// returned.
func (db *PreparedStmtDB) BeginTx(ctx context.Context, opt *sql.TxOptions) (ConnPool, error) {
	if beginner, ok := db.ConnPool.(TxBeginner); ok {
		tx, err := beginner.BeginTx(ctx, opt)
		return &PreparedStmtTX{PreparedStmtDB: db, Tx: tx}, err
	}

	beginner, ok := db.ConnPool.(ConnPoolBeginner)
	if !ok {
		return nil, ErrInvalidTransaction
	}

	connPool, err := beginner.BeginTx(ctx, opt)
	if err != nil {
		return nil, err
	}
	if tx, ok := connPool.(Tx); ok {
		return &PreparedStmtTX{PreparedStmtDB: db, Tx: tx}, nil
	}
	return nil, ErrInvalidTransaction
}

// ExecContext prepares or reuses query and executes it with bound args. It returns
// preparation/execution errors and evicts the entry when execution reports
// driver.ErrBadConn. It does not retry the operation itself.
func (db *PreparedStmtDB) ExecContext(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error) {
	stmt, err := db.prepare(ctx, db.ConnPool, false, query)
	if err == nil {
		result, err = stmt.ExecContext(ctx, args...)
		if errors.Is(err, driver.ErrBadConn) {
			db.Stmts.Delete(query)
		}
	}
	return result, err
}

// QueryContext prepares or reuses query and returns rows owned by the caller. Close the
// rows and check iteration errors. It evicts the cached entry on driver.ErrBadConn
// returned during query execution.
func (db *PreparedStmtDB) QueryContext(ctx context.Context, query string, args ...interface{}) (rows *sql.Rows, err error) {
	stmt, err := db.prepare(ctx, db.ConnPool, false, query)
	if err == nil {
		rows, err = stmt.QueryContext(ctx, args...)
		if errors.Is(err, driver.ErrBadConn) {
			db.Stmts.Delete(query)
		}
	}
	return rows, err
}

// QueryRowContext executes query through a cached statement, deferring normal query
// errors to Row.Scan. If preparation fails, this implementation returns an empty
// sql.Row and loses the preparation error. That row is not safe to Scan and may panic;
// use QueryContext to receive preparation errors directly.
func (db *PreparedStmtDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	stmt, err := db.prepare(ctx, db.ConnPool, false, query)
	if err == nil {
		return stmt.QueryRowContext(ctx, args...)
	}
	return &sql.Row{}
}

// Ping checks the underlying shared *sql.DB, returning ErrInvalidDB if no pool can be
// exposed. It does not validate each cached statement.
func (db *PreparedStmtDB) Ping() error {
	conn, err := db.GetDBConn()
	if err != nil {
		return err
	}
	return conn.Ping()
}

// PreparedStmtTX executes cached statements on a specific transaction using
// StmtContext. It shares the cache with PreparedStmtDB but completes only its own Tx.
// Do not retain it after Commit or Rollback.
type PreparedStmtTX struct {
	Tx
	// PreparedStmtDB supplies the shared statement store and access to the originating
	// connection pool.
	PreparedStmtDB *PreparedStmtDB
}

// GetDBConn returns the originating shared pool. Operations performed directly on that
// pool do not participate in this transaction.
func (db *PreparedStmtTX) GetDBConn() (*sql.DB, error) {
	return db.PreparedStmtDB.GetDBConn()
}

// Commit completes the wrapped transaction or returns ErrInvalidTransaction when its
// transaction value is nil. It does not close the shared statement cache.
func (tx *PreparedStmtTX) Commit() error {
	if tx.Tx != nil && !reflect.ValueOf(tx.Tx).IsNil() {
		return tx.Tx.Commit()
	}
	return ErrInvalidTransaction
}

// Rollback aborts the wrapped transaction or returns ErrInvalidTransaction when its
// transaction value is nil. It does not close the shared statement cache.
func (tx *PreparedStmtTX) Rollback() error {
	if tx.Tx != nil && !reflect.ValueOf(tx.Tx).IsNil() {
		return tx.Tx.Rollback()
	}
	return ErrInvalidTransaction
}

// ExecContext prepares or reuses query, binds the statement to this transaction, and
// executes args. A driver.ErrBadConn result evicts the shared cache entry; no
// application-level retry is performed.
func (tx *PreparedStmtTX) ExecContext(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error) {
	stmt, err := tx.PreparedStmtDB.prepare(ctx, tx.Tx, true, query)
	if err == nil {
		result, err = tx.Tx.StmtContext(ctx, stmt.Stmt).ExecContext(ctx, args...)
		if errors.Is(err, driver.ErrBadConn) {
			tx.PreparedStmtDB.Stmts.Delete(query)
		}
	}
	return result, err
}

// QueryContext executes a cached statement bound to this transaction. The caller owns
// returned rows and must close them before completing the transaction. A
// driver.ErrBadConn query error evicts the shared entry.
func (tx *PreparedStmtTX) QueryContext(ctx context.Context, query string, args ...interface{}) (rows *sql.Rows, err error) {
	stmt, err := tx.PreparedStmtDB.prepare(ctx, tx.Tx, true, query)
	if err == nil {
		rows, err = tx.Tx.StmtContext(ctx, stmt.Stmt).QueryContext(ctx, args...)
		if errors.Is(err, driver.ErrBadConn) {
			tx.PreparedStmtDB.Stmts.Delete(query)
		}
	}
	return rows, err
}

// QueryRowContext binds a cached statement to this transaction and returns its
// single-row result. If preparation fails, this implementation returns an empty sql.Row
// and loses the preparation error. That row is not safe to Scan and may panic; use
// QueryContext for direct preparation-error reporting.
func (tx *PreparedStmtTX) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	stmt, err := tx.PreparedStmtDB.prepare(ctx, tx.Tx, true, query)
	if err == nil {
		return tx.Tx.StmtContext(ctx, stmt.Stmt).QueryRowContext(ctx, args...)
	}
	return &sql.Row{}
}

// Ping checks the originating pool through GetDBConn. It does not prove that this
// transaction is still active or usable.
func (tx *PreparedStmtTX) Ping() error {
	conn, err := tx.GetDBConn()
	if err != nil {
		return err
	}
	return conn.Ping()
}
