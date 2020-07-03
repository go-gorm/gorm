package gorm

import (
	"context"
	"database/sql"
	"sync"
)

type PreparedStmtDB struct {
	Stmts map[string]*sql.Stmt
	mux   sync.RWMutex
	ConnPool
}

func (db *PreparedStmtDB) Close() {
	db.mux.Lock()
	for k, stmt := range db.Stmts {
		delete(db.Stmts, k)
		stmt.Close()
	}

	db.mux.Unlock()
}

func (db *PreparedStmtDB) prepare(query string) (*sql.Stmt, error) {
	db.mux.RLock()
	if stmt, ok := db.Stmts[query]; ok {
		db.mux.RUnlock()
		return stmt, nil
	}
	db.mux.RUnlock()

	db.mux.Lock()
	// double check
	if stmt, ok := db.Stmts[query]; ok {
		db.mux.Unlock()
		return stmt, nil
	}

	stmt, err := db.ConnPool.PrepareContext(context.Background(), query)
	if err == nil {
		db.Stmts[query] = stmt
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

func (db *PreparedStmtDB) ExecContext(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error) {
	stmt, err := db.prepare(query)
	if err == nil {
		result, err = stmt.ExecContext(ctx, args...)
		if err != nil {
			db.mux.Lock()
			stmt.Close()
			delete(db.Stmts, query)
			db.mux.Unlock()
		}
	}
	return result, err
}

func (db *PreparedStmtDB) QueryContext(ctx context.Context, query string, args ...interface{}) (rows *sql.Rows, err error) {
	stmt, err := db.prepare(query)
	if err == nil {
		rows, err = stmt.QueryContext(ctx, args...)
		if err != nil {
			db.mux.Lock()
			stmt.Close()
			delete(db.Stmts, query)
			db.mux.Unlock()
		}
	}
	return rows, err
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

func (tx *PreparedStmtTX) ExecContext(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error) {
	stmt, err := tx.PreparedStmtDB.prepare(query)
	if err == nil {
		result, err = tx.Tx.Stmt(stmt).ExecContext(ctx, args...)
		if err != nil {
			tx.PreparedStmtDB.mux.Lock()
			stmt.Close()
			delete(tx.PreparedStmtDB.Stmts, query)
			tx.PreparedStmtDB.mux.Unlock()
		}
	}
	return result, err
}

func (tx *PreparedStmtTX) QueryContext(ctx context.Context, query string, args ...interface{}) (rows *sql.Rows, err error) {
	stmt, err := tx.PreparedStmtDB.prepare(query)
	if err == nil {
		rows, err = tx.Tx.Stmt(stmt).QueryContext(ctx, args...)
		if err != nil {
			tx.PreparedStmtDB.mux.Lock()
			stmt.Close()
			delete(tx.PreparedStmtDB.Stmts, query)
			tx.PreparedStmtDB.mux.Unlock()
		}
	}
	return rows, err
}

func (tx *PreparedStmtTX) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	stmt, err := tx.PreparedStmtDB.prepare(query)
	if err == nil {
		return tx.Tx.Stmt(stmt).QueryRowContext(ctx, args...)
	}
	return &sql.Row{}
}
