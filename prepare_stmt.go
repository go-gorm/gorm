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

type PreparedStmtDB struct {
	Stmts stmt_store.Store
	Mux   *sync.RWMutex
	ConnPool
}

func NewPreparedStmtDB(connPool ConnPool, maxSize int, ttl time.Duration) *PreparedStmtDB {
	return &PreparedStmtDB{
		ConnPool: connPool,
		Stmts:    stmt_store.New(maxSize, ttl),
		Mux:      &sync.RWMutex{},
	}
}

func (db *PreparedStmtDB) GetDBConn() (*sql.DB, error) {
	if sqldb, ok := db.ConnPool.(*sql.DB); ok {
		return sqldb, nil
	}

	if dbConnector, ok := db.ConnPool.(GetDBConnector); ok && dbConnector != nil {
		return dbConnector.GetDBConn()
	}

	return nil, ErrInvalidDB
}

func (db *PreparedStmtDB) Close() {
	db.Mux.Lock()
	defer db.Mux.Unlock()
	if db.Stmts == nil {
		return
	}

	for _, stmt := range db.Stmts.AllMap() {
		go stmt.Close()
	}
	// setting db.Stmts to nil to avoid further using
	db.Stmts = nil
}

func (sdb *PreparedStmtDB) Reset() {
	sdb.Mux.Lock()
	defer sdb.Mux.Unlock()
	if sdb.Stmts == nil {
		return
	}
	for _, stmt := range sdb.Stmts.AllMap() {
		go stmt.Close()
	}

	// Migrator
	sdb.Stmts = stmt_store.New(0, 0)
}

func (db *PreparedStmtDB) prepare(ctx context.Context, conn ConnPool, isTransaction bool, query string) (stmt_store.Stmt, error) {
	db.Mux.RLock()
	if db.Stmts != nil {
		if stmt, ok := db.Stmts.Get(query); ok && (!stmt.Transaction || isTransaction) {
			db.Mux.RUnlock()
			if err := stmt.Error(); err != nil {
				return stmt_store.Stmt{}, err
			}

			return *stmt, nil
		}
	}
	db.Mux.RUnlock()

	db.Mux.Lock()
	if db.Stmts != nil {
		// double check
		if stmt, ok := db.Stmts.Get(query); ok && (!stmt.Transaction || isTransaction) {
			db.Mux.Unlock()
			if err := stmt.Error(); err != nil {
				return stmt_store.Stmt{}, err
			}

			return *stmt, nil
		}
	}
	// check db.Stmts first to avoid Segmentation Fault(setting value to nil map)
	// which cause by calling Close and executing SQL concurrently
	if db.Stmts == nil {
		db.Mux.Unlock()
		return stmt_store.Stmt{}, ErrInvalidDB
	}
	// cache preparing stmt first
	cacheStmt := stmt_store.NewStmt(isTransaction)
	db.Stmts.Set(query, cacheStmt)
	db.Mux.Unlock()

	// prepare completed
	defer cacheStmt.Done()

	// Reason why cannot lock conn.PrepareContext
	// suppose the maxopen is 1, g1 is creating record and g2 is querying record.
	// 1. g1 begin tx, g1 is requeue because of waiting for the system call, now `db.ConnPool` db.numOpen == 1.
	// 2. g2 select lock `conn.PrepareContext(ctx, query)`, now db.numOpen == db.maxOpen , wait for release.
	// 3. g1 tx exec insert, wait for unlock `conn.PrepareContext(ctx, query)` to finish tx and release.
	stmt, err := conn.PrepareContext(ctx, query)
	if err != nil {
		cacheStmt.AddError(err)
		db.Mux.Lock()
		db.Stmts.Delete(query)
		db.Mux.Unlock()
		return stmt_store.Stmt{}, err
	}

	db.Mux.Lock()
	cacheStmt.Stmt = stmt
	db.Mux.Unlock()

	return *cacheStmt, nil
}

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

func (db *PreparedStmtDB) ExecContext(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error) {
	stmt, err := db.prepare(ctx, db.ConnPool, false, query)
	if err == nil {
		result, err = stmt.ExecContext(ctx, args...)
		if errors.Is(err, driver.ErrBadConn) {
			db.Mux.Lock()
			defer db.Mux.Unlock()
			go stmt.Close()
			db.Stmts.Delete(query)
		}
	}
	return result, err
}

func (db *PreparedStmtDB) QueryContext(ctx context.Context, query string, args ...interface{}) (rows *sql.Rows, err error) {
	stmt, err := db.prepare(ctx, db.ConnPool, false, query)
	if err == nil {
		rows, err = stmt.QueryContext(ctx, args...)
		if errors.Is(err, driver.ErrBadConn) {
			db.Mux.Lock()
			defer db.Mux.Unlock()

			go stmt.Close()
			db.Stmts.Delete(query)
		}
	}
	return rows, err
}

func (db *PreparedStmtDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	stmt, err := db.prepare(ctx, db.ConnPool, false, query)
	if err == nil {
		return stmt.QueryRowContext(ctx, args...)
	}
	return &sql.Row{}
}

func (db *PreparedStmtDB) Ping() error {
	conn, err := db.GetDBConn()
	if err != nil {
		return err
	}
	return conn.Ping()
}

type PreparedStmtTX struct {
	Tx
	PreparedStmtDB *PreparedStmtDB
}

func (db *PreparedStmtTX) GetDBConn() (*sql.DB, error) {
	return db.PreparedStmtDB.GetDBConn()
}

func (tx *PreparedStmtTX) Commit() error {
	if tx.Tx != nil && !reflect.ValueOf(tx.Tx).IsNil() {
		return tx.Tx.Commit()
	}
	return ErrInvalidTransaction
}

func (tx *PreparedStmtTX) Rollback() error {
	if tx.Tx != nil && !reflect.ValueOf(tx.Tx).IsNil() {
		return tx.Tx.Rollback()
	}
	return ErrInvalidTransaction
}

func (tx *PreparedStmtTX) ExecContext(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error) {
	stmt, err := tx.PreparedStmtDB.prepare(ctx, tx.Tx, true, query)
	if err == nil {
		result, err = tx.Tx.StmtContext(ctx, stmt.Stmt).ExecContext(ctx, args...)
		if errors.Is(err, driver.ErrBadConn) {
			tx.PreparedStmtDB.Mux.Lock()
			defer tx.PreparedStmtDB.Mux.Unlock()

			go stmt.Close()
			tx.PreparedStmtDB.Stmts.Delete(query)
		}
	}
	return result, err
}

func (tx *PreparedStmtTX) QueryContext(ctx context.Context, query string, args ...interface{}) (rows *sql.Rows, err error) {
	stmt, err := tx.PreparedStmtDB.prepare(ctx, tx.Tx, true, query)
	if err == nil {
		rows, err = tx.Tx.StmtContext(ctx, stmt.Stmt).QueryContext(ctx, args...)
		if errors.Is(err, driver.ErrBadConn) {
			tx.PreparedStmtDB.Mux.Lock()
			defer tx.PreparedStmtDB.Mux.Unlock()

			go stmt.Close()
			tx.PreparedStmtDB.Stmts.Delete(query)
		}
	}
	return rows, err
}

func (tx *PreparedStmtTX) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	stmt, err := tx.PreparedStmtDB.prepare(ctx, tx.Tx, true, query)
	if err == nil {
		return tx.Tx.StmtContext(ctx, stmt.Stmt).QueryRowContext(ctx, args...)
	}
	return &sql.Row{}
}

func (tx *PreparedStmtTX) Ping() error {
	conn, err := tx.GetDBConn()
	if err != nil {
		return err
	}
	return conn.Ping()
}
