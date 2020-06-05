package gorm

import (
	"context"
	"database/sql"
	"sync"
)

type PreparedStmtDB struct {
	stmts map[string]*sql.Stmt
	mux   sync.RWMutex
	ConnPool
}

func (db *PreparedStmtDB) prepare(query string) (*sql.Stmt, error) {
	db.mux.RLock()
	if stmt, ok := db.stmts[query]; ok {
		db.mux.RUnlock()
		return stmt, nil
	}
	db.mux.RUnlock()

	db.mux.Lock()
	stmt, err := db.ConnPool.PrepareContext(context.Background(), query)
	if err == nil {
		db.stmts[query] = stmt
	}
	db.mux.Unlock()

	return stmt, err
}

func (db *PreparedStmtDB) BeginTx(ctx context.Context, opt *sql.TxOptions) (ConnPool, error) {
	if beginner, ok := db.ConnPool.(TxBeginner); ok {
		tx, err := beginner.BeginTx(ctx, opt)
		return &PreparedStmtTX{PreparedStmtDB: db, Tx: tx}, err
	}
	return nil, ErrInvalidTransaction
}

func (db *PreparedStmtDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	stmt, err := db.prepare(query)
	if err == nil {
		return stmt.ExecContext(ctx, args...)
	}
	return nil, err
}

func (db *PreparedStmtDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	stmt, err := db.prepare(query)
	if err == nil {
		return stmt.QueryContext(ctx, args...)
	}
	return nil, err
}

func (db *PreparedStmtDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	stmt, err := db.prepare(query)
	if err == nil {
		return stmt.QueryRowContext(ctx, args...)
	}
	return &sql.Row{}
}

type PreparedStmtTX struct {
	*sql.Tx
	PreparedStmtDB *PreparedStmtDB
}

func (tx *PreparedStmtTX) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	stmt, err := tx.PreparedStmtDB.prepare(query)
	if err == nil {
		return tx.Tx.Stmt(stmt).ExecContext(ctx, args...)
	}
	return nil, err
}

func (tx *PreparedStmtTX) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	stmt, err := tx.PreparedStmtDB.prepare(query)
	if err == nil {
		return tx.Tx.Stmt(stmt).QueryContext(ctx, args...)
	}
	return nil, err
}

func (tx *PreparedStmtTX) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	stmt, err := tx.PreparedStmtDB.prepare(query)
	if err == nil {
		return tx.Tx.Stmt(stmt).QueryRowContext(ctx, args...)
	}
	return &sql.Row{}
}
