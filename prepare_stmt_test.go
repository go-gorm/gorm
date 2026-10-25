package gorm_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// failPreparePool wraps a real ConnPool but makes every PrepareContext fail,
// so the PreparedStmtDB prepare path is forced to fail.
type failPreparePool struct {
	gorm.ConnPool
	err error
}

func (p failPreparePool) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return nil, p.err
}

// failPrepareTx is the transaction counterpart of failPreparePool.
type failPrepareTx struct {
	gorm.Tx
	err error
}

func (t failPrepareTx) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return nil, t.err
}

// When the statement cannot be prepared, QueryRowContext used to return a
// zero-value *sql.Row whose Scan panics with a nil pointer dereference and
// swallows the error. It must instead return a row that reports the failure
// from Scan. See BUG H02.
func TestPreparedStmtQueryRowPrepareFailure(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db failed: %v", err)
	}

	prepareErr := errors.New("forced prepare failure")
	pdb := gorm.NewPreparedStmtDB(failPreparePool{ConnPool: db.Statement.ConnPool, err: prepareErr}, 0, 0)

	t.Run("query error surfaces from Scan instead of panicking", func(t *testing.T) {
		tx := db.Raw("SELECT * FROM prepared_stmt_missing_table")
		tx.Statement.ConnPool = pdb

		var count int
		row := tx.Row()
		if err := row.Scan(&count); err == nil {
			t.Fatal("expected an error from Scan on invalid SQL, got nil")
		}
	})

	t.Run("valid query falls back to direct execution", func(t *testing.T) {
		tx := db.Raw("SELECT 42")
		tx.Statement.ConnPool = pdb

		var got int
		row := tx.Row()
		if err := row.Scan(&got); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 42 {
			t.Fatalf("expected 42, got %d", got)
		}
	})
}

func TestPreparedStmtTxQueryRowPrepareFailure(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db failed: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db failed: %v", err)
	}
	sqlTx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin tx failed: %v", err)
	}
	defer func() { _ = sqlTx.Rollback() }()

	prepareErr := errors.New("forced prepare failure")
	pdb := gorm.NewPreparedStmtDB(failPreparePool{ConnPool: db.Statement.ConnPool, err: prepareErr}, 0, 0)
	ptx := &gorm.PreparedStmtTX{PreparedStmtDB: pdb, Tx: failPrepareTx{Tx: sqlTx, err: prepareErr}}

	t.Run("query error surfaces from Scan instead of panicking", func(t *testing.T) {
		var count int
		row := ptx.QueryRowContext(ctx, "SELECT * FROM prepared_stmt_missing_table")
		if err := row.Scan(&count); err == nil {
			t.Fatal("expected an error from Scan on invalid SQL, got nil")
		}
	})

	t.Run("valid query falls back to direct execution", func(t *testing.T) {
		var got int
		row := ptx.QueryRowContext(ctx, "SELECT 43")
		if err := row.Scan(&got); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 43 {
			t.Fatalf("expected 43, got %d", got)
		}
	})
}
