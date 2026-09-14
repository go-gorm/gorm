package gorm_test

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"gorm.io/gorm"
	"gorm.io/gorm/internal/stmt_store"
)

// evictingStmtStore reproduces eviction after a cache lookup, before its caller
// can use the returned statement. Closing explicitly waits for the asynchronous
// eviction callback's effect without relying on goroutine scheduling.
type evictingStmtStore struct {
	stmt_store.Store
	evictNext   bool
	evictNew    bool
	evictedStmt *stmt_store.Stmt
}

func (s *evictingStmtStore) Get(query string) (*stmt_store.Stmt, bool) {
	stmt, ok := s.Store.Get(query)
	if ok && s.evictNext {
		s.evictNext = false
		s.evict(query, stmt)
	}
	return stmt, ok
}

func (s *evictingStmtStore) New(ctx context.Context, query string, isTransaction bool, conn stmt_store.ConnPool, locker sync.Locker) (*stmt_store.Stmt, error) {
	stmt, err := s.Store.New(ctx, query, isTransaction, conn, locker)
	if err == nil && s.evictNew {
		s.evictNew = false
		s.evict(query, stmt)
	}
	return stmt, err
}

func (s *evictingStmtStore) evict(query string, stmt *stmt_store.Stmt) {
	s.Delete(query)
	_ = stmt.Close()
	s.evictedStmt = stmt
}

func TestPreparedStmtEviction(t *testing.T) {
	for _, eviction := range []string{"cached", "new"} {
		for _, transaction := range []bool{false, true} {
			for _, operation := range []string{"Exec", "Query", "QueryRow"} {
				name := fmt.Sprintf("%s/%s/transaction=%t", eviction, operation, transaction)
				t.Run(name, func(t *testing.T) {
					testPreparedStmtEviction(t, eviction, transaction, operation)
				})
			}
		}
	}
}

func testPreparedStmtEviction(t *testing.T, eviction string, transaction bool, operation string) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	prepared := gorm.NewPreparedStmtDB(db, 1, time.Hour)
	defer prepared.Close()
	store := &evictingStmtStore{Store: prepared.Stmts}
	prepared.Stmts = store
	ctx := context.Background()
	if eviction == "cached" {
		if _, err := prepared.ExecContext(ctx, "SELECT 1"); err != nil {
			t.Fatal(err)
		}
		store.evictNext = true
	} else {
		store.evictNew = true
	}
	var conn gorm.ConnPool = prepared
	if transaction {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback() }()
		conn = &gorm.PreparedStmtTX{Tx: tx, PreparedStmtDB: prepared}
	}
	if err := executePreparedQuery(ctx, conn, operation, "SELECT 1"); err != nil {
		t.Fatalf("cache eviction must not fail an in-flight request: %v", err)
	}
	if store.evictedStmt == nil {
		t.Fatal("test did not evict a statement")
	}
	if _, err := store.evictedStmt.ExecContext(ctx); err == nil {
		t.Fatal("evicted statement was not closed after execution")
	}
}

func executePreparedQuery(ctx context.Context, conn gorm.ConnPool, operation, query string) error {
	var value int
	switch operation {
	case "Exec":
		_, err := conn.ExecContext(ctx, query)
		return err
	case "Query":
		rows, err := conn.QueryContext(ctx, query)
		if err != nil {
			return err
		}
		defer func() { _ = rows.Close() }()
		if !rows.Next() {
			return fmt.Errorf("expected a result row, got error %v", rows.Err())
		}
		if err := rows.Scan(&value); err != nil {
			return err
		}
	case "QueryRow":
		if err := conn.QueryRowContext(ctx, query).Scan(&value); err != nil {
			return err
		}
	}
	if value != 1 {
		return fmt.Errorf("query returned %d, want 1", value)
	}
	return nil
}

func TestPreparedStmtConcurrentEviction(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	prepared := gorm.NewPreparedStmtDB(db, 1, time.Hour)
	defer prepared.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			query := fmt.Sprintf("SELECT 1 /* worker %d */", i)
			for j := 0; j < 200; j++ {
				for _, operation := range []string{"Exec", "Query", "QueryRow"} {
					if err := executePreparedQuery(ctx, prepared, operation, query); err != nil {
						t.Errorf("%s failed during concurrent cache eviction: %v", operation, err)
						return
					}
				}
			}
		}(i)
	}
	wg.Wait()
}
