package gorm

import (
	"context"
	"database/sql"
	"sync"
)

type PreparedStmtDB struct {
	Stmts       map[string]*sql.Stmt
	PreparedSQL []string
	Mux         *sync.RWMutex
	ConnPool
}

func (db *PreparedStmtDB) Close() {
	db.Mux.Lock()
	for _, query := range db.PreparedSQL {
		if stmt, ok := db.Stmts[query]; ok {
			delete(db.Stmts, query)
			stmt.Close()
		}
	}

	db.Mux.Unlock()
}

func (db *PreparedStmtDB) prepare(ctx context.Context, query string) (*sql.Stmt, error) {
	db.Mux.RLock()
	if stmt, ok := db.Stmts[query]; ok {
		db.Mux.RUnlock()
		return stmt, nil
	}
	db.Mux.RUnlock()

	db.Mux.Lock()
	// double check
	if stmt, ok := db.Stmts[query]; ok {
		db.Mux.Unlock()
		return stmt, nil
	}

	stmt, err := db.ConnPool.PrepareContext(ctx, query)
	if err == nil {
		db.Stmts[query] = stmt
		db.PreparedSQL = append(db.PreparedSQL, query)
	}
	db.Mux.Unlock()

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
	stmt, err := db.prepare(ctx, query)
	if err == nil {
		result, err = stmt.ExecContext(ctx, args...)
		if err != nil {
			db.Mux.Lock()
			stmt.Close()
			delete(db.Stmts, query)
			db.Mux.Unlock()
		}
	}
	return result, err
}

func (db *PreparedStmtDB) QueryContext(ctx context.Context, query string, args ...interface{}) (rows *sql.Rows, err error) {
	stmt, err := db.prepare(ctx, query)
	if err == nil {
		rows, err = stmt.QueryContext(ctx, args...)
		if err != nil {
			db.Mux.Lock()
			stmt.Close()
			delete(db.Stmts, query)
			db.Mux.Unlock()
		}
	}
	return rows, err
}

func (db *PreparedStmtDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	stmt, err := db.prepare(ctx, query)
	if err == nil {
		return stmt.QueryRowContext(ctx, args...)
	}
	return &sql.Row{}
}

type PreparedStmtTX struct {
	*sql.Tx
	PreparedStmtDB *PreparedStmtDB
}

func (tx *PreparedStmtTX) Commit() error {
	if tx.Tx != nil {
		return tx.Tx.Commit()
	}
	return ErrInvalidTransaction
}

func (tx *PreparedStmtTX) Rollback() error {
	if tx.Tx != nil {
		return tx.Tx.Rollback()
	}
	return ErrInvalidTransaction
}

func (tx *PreparedStmtTX) ExecContext(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error) {
	stmt, err := tx.PreparedStmtDB.prepare(ctx, query)
	if err == nil {
		result, err = tx.Tx.StmtContext(ctx, stmt).ExecContext(ctx, args...)
		if err != nil {
			tx.PreparedStmtDB.Mux.Lock()
			stmt.Close()
			delete(tx.PreparedStmtDB.Stmts, query)
			tx.PreparedStmtDB.Mux.Unlock()
		}
	}
	return result, err
}

func (tx *PreparedStmtTX) QueryContext(ctx context.Context, query string, args ...interface{}) (rows *sql.Rows, err error) {
	stmt, err := tx.PreparedStmtDB.prepare(ctx, query)
	if err == nil {
		rows, err = tx.Tx.Stmt(stmt).QueryContext(ctx, args...)
		if err != nil {
			tx.PreparedStmtDB.Mux.Lock()
			stmt.Close()
			delete(tx.PreparedStmtDB.Stmts, query)
			tx.PreparedStmtDB.Mux.Unlock()
		}
	}
	return rows, err
}

func (tx *PreparedStmtTX) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	stmt, err := tx.PreparedStmtDB.prepare(ctx, query)
	if err == nil {
		return tx.Tx.StmtContext(ctx, stmt).QueryRowContext(ctx, args...)
	}
	return &sql.Row{}
}
