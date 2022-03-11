package gorm

import (
	"context"
	"database/sql"
	"sync"
)

type Stmt struct {
	*sql.Stmt
	Transaction bool
}

type PreparedStmtDB struct {
	Stmts       map[string]Stmt
	PreparedSQL []string
	Mux         *sync.RWMutex
	ConnPool
}

func (db *PreparedStmtDB) GetDBConn() (*sql.DB, error) {
	if dbConnector, ok := db.ConnPool.(GetDBConnector); ok && dbConnector != nil {
		return dbConnector.GetDBConn()
	}

	if sqldb, ok := db.ConnPool.(*sql.DB); ok {
		return sqldb, nil
	}

	return nil, ErrInvalidDB
}

func (db *PreparedStmtDB) Close() {
	db.Mux.Lock()
	defer db.Mux.Unlock()

	for _, query := range db.PreparedSQL {
		if stmt, ok := db.Stmts[query]; ok {
			delete(db.Stmts, query)
			go stmt.Close()
		}
	}
}

func (db *PreparedStmtDB) prepare(ctx context.Context, conn ConnPool, isTransaction bool, query string) (Stmt, error) {
	db.Mux.RLock()
	if stmt, ok := db.Stmts[query]; ok && (!stmt.Transaction || isTransaction) {
		db.Mux.RUnlock()
		return stmt, nil
	}
	db.Mux.RUnlock()

	db.Mux.Lock()
	defer db.Mux.Unlock()

	// double check
	if stmt, ok := db.Stmts[query]; ok && (!stmt.Transaction || isTransaction) {
		return stmt, nil
	} else if ok {
		go stmt.Close()
	}

	stmt, err := conn.PrepareContext(ctx, query)
	if err == nil {
		db.Stmts[query] = Stmt{Stmt: stmt, Transaction: isTransaction}
		db.PreparedSQL = append(db.PreparedSQL, query)
	}

	return db.Stmts[query], err
}

func (db *PreparedStmtDB) BeginTx(ctx context.Context, opt *sql.TxOptions) (ConnPool, error) {
	if beginner, ok := db.ConnPool.(TxBeginner); ok {
		tx, err := beginner.BeginTx(ctx, opt)
		return &PreparedStmtTX{PreparedStmtDB: db, Tx: tx}, err
	}
	return nil, ErrInvalidTransaction
}

func (db *PreparedStmtDB) ExecContext(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error) {
	stmt, err := db.prepare(ctx, db.ConnPool, false, query)
	if err == nil {
		result, err = stmt.ExecContext(ctx, args...)
		if err != nil {
			db.Mux.Lock()
			defer db.Mux.Unlock()
			go stmt.Close()
			delete(db.Stmts, query)
		}
	}
	return result, err
}

func (db *PreparedStmtDB) QueryContext(ctx context.Context, query string, args ...interface{}) (rows *sql.Rows, err error) {
	stmt, err := db.prepare(ctx, db.ConnPool, false, query)
	if err == nil {
		rows, err = stmt.QueryContext(ctx, args...)
		if err != nil {
			db.Mux.Lock()
			defer db.Mux.Unlock()

			go stmt.Close()
			delete(db.Stmts, query)
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

type PreparedStmtTX struct {
	Tx
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
	stmt, err := tx.PreparedStmtDB.prepare(ctx, tx.Tx, true, query)
	if err == nil {
		result, err = tx.Tx.StmtContext(ctx, stmt.Stmt).ExecContext(ctx, args...)
		if err != nil {
			tx.PreparedStmtDB.Mux.Lock()
			defer tx.PreparedStmtDB.Mux.Unlock()

			go stmt.Close()
			delete(tx.PreparedStmtDB.Stmts, query)
		}
	}
	return result, err
}

func (tx *PreparedStmtTX) QueryContext(ctx context.Context, query string, args ...interface{}) (rows *sql.Rows, err error) {
	stmt, err := tx.PreparedStmtDB.prepare(ctx, tx.Tx, true, query)
	if err == nil {
		rows, err = tx.Tx.StmtContext(ctx, stmt.Stmt).QueryContext(ctx, args...)
		if err != nil {
			tx.PreparedStmtDB.Mux.Lock()
			defer tx.PreparedStmtDB.Mux.Unlock()

			go stmt.Close()
			delete(tx.PreparedStmtDB.Stmts, query)
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
