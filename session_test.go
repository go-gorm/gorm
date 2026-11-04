package gorm

import (
	"context"
	"database/sql"
	"testing"
)

type sessionConnPool struct{}

func (p *sessionConnPool) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	return nil, nil
}

func (p *sessionConnPool) ExecContext(context.Context, string, ...interface{}) (sql.Result, error) {
	return nil, nil
}

func (p *sessionConnPool) QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func (p *sessionConnPool) QueryRowContext(context.Context, string, ...interface{}) *sql.Row {
	return nil
}

func (p *sessionConnPool) BeginTx(context.Context, *sql.TxOptions) (ConnPool, error) {
	return &sessionTx{}, nil
}

type sessionTx struct {
	sessionConnPool
}

func (tx *sessionTx) Commit() error {
	return nil
}

func (tx *sessionTx) Rollback() error {
	return nil
}

func (tx *sessionTx) StmtContext(context.Context, *sql.Stmt) *sql.Stmt {
	return nil
}

func TestSessionDisablePrepareStmt(t *testing.T) {
	pool := &sessionConnPool{}
	db, err := Open(nil, &Config{
		ConnPool:             pool,
		PrepareStmt:          true,
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	if _, ok := db.Statement.ConnPool.(*PreparedStmtDB); !ok {
		t.Fatalf("expected parent session to use prepared statements")
	}

	tx := db.Session(&Session{DisablePrepareStmt: true})
	if tx.Statement.ConnPool != pool {
		t.Fatalf("expected unwrapped connection pool, got %T", tx.Statement.ConnPool)
	}
	if tx.ConnPool != pool {
		t.Fatalf("expected config connection pool to be unwrapped, got %T", tx.ConnPool)
	}
	if tx.PrepareStmt {
		t.Fatalf("expected prepared statement mode to be disabled")
	}

	if _, ok := db.Statement.ConnPool.(*PreparedStmtDB); !ok {
		t.Fatalf("parent session should still use prepared statements")
	}
}

func TestTransactionSessionDisablePrepareStmt(t *testing.T) {
	db, err := Open(nil, &Config{
		ConnPool:             &sessionConnPool{},
		PrepareStmt:          true,
		DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("failed to begin transaction: %v", tx.Error)
	}
	if _, ok := tx.Statement.ConnPool.(*PreparedStmtTX); !ok {
		t.Fatalf("expected prepared transaction, got %T", tx.Statement.ConnPool)
	}

	plainTx := tx.Session(&Session{DisablePrepareStmt: true})
	if _, ok := plainTx.Statement.ConnPool.(*sessionTx); !ok {
		t.Fatalf("expected unwrapped transaction, got %T", plainTx.Statement.ConnPool)
	}
	if plainTx.ConnPool != plainTx.Statement.ConnPool {
		t.Fatalf("expected config and statement connection pools to match")
	}
	if plainTx.PrepareStmt {
		t.Fatalf("expected prepared statement mode to be disabled")
	}
}
