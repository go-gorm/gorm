package stmt_store

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func openStmtTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestStmtCloseWaitsForActiveUsers(t *testing.T) {
	db := openStmtTestDB(t)
	store := New(1, time.Hour)
	var mu sync.Mutex
	mu.Lock()
	stmt, err := store.New(context.Background(), "SELECT 1", false, db, &mu)
	if err != nil {
		t.Fatal(err)
	}
	if !stmt.Acquire() {
		t.Fatal("could not acquire a live statement")
	}
	store.Delete("SELECT 1")
	if err := stmt.Close(); err != nil {
		t.Fatal(err)
	}
	if stmt.Acquire() {
		t.Fatal("acquired a statement after close")
	}
	stmt.Release()
	if _, err := stmt.ExecContext(context.Background()); err != nil {
		t.Fatalf("close invalidated a statement still in use: %v", err)
	}
	stmt.Release()
	if _, err := stmt.ExecContext(context.Background()); err == nil {
		t.Fatal("statement remained open after the final release")
	}
}

type blockedPrepareConn struct {
	*sql.DB
	started chan struct{}
	resume  chan struct{}
	err     error
}

func (c *blockedPrepareConn) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	close(c.started)
	select {
	case <-c.resume:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if c.err != nil {
		return nil, c.err
	}
	return c.DB.PrepareContext(ctx, query)
}

func TestStmtEvictionDuringPreparation(t *testing.T) {
	for _, fail := range []bool{false, true} {
		name := "success"
		if fail {
			name = "failure"
		}
		t.Run(name, func(t *testing.T) {
			testStmtEvictionDuringPreparation(t, fail)
		})
	}
}

func testStmtEvictionDuringPreparation(t *testing.T, fail bool) {
	t.Helper()
	db := openStmtTestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn := &blockedPrepareConn{DB: db, started: make(chan struct{}), resume: make(chan struct{})}
	if fail {
		conn.err = errors.New("preparation failed")
	}
	store := New(1, time.Hour).(*lruStore)
	done := make(chan prepareResult, 1)
	go func() {
		var mu sync.Mutex
		mu.Lock()
		stmt, err := store.New(ctx, "SELECT 1", false, conn, &mu)
		done <- prepareResult{stmt, err}
	}()
	select {
	case <-conn.started:
	case <-ctx.Done():
		t.Fatal("preparation did not start")
	}
	// Inspect the published entry without Get, which waits for preparation.
	pending, ok := store.lru.Peek("SELECT 1")
	if !ok {
		t.Fatal("statement was not published before preparation")
	}
	store.Delete("SELECT 1")
	close(conn.resume)
	got := waitForPreparation(t, ctx, done)
	if !errors.Is(got.err, conn.err) {
		t.Fatalf("preparation error = %v, want %v", got.err, conn.err)
	}
	if err := pending.Close(); err != nil {
		t.Fatal(err)
	}
	if !fail {
		if _, err := got.stmt.ExecContext(ctx); err != nil {
			t.Fatalf("eviction invalidated the creator's statement: %v", err)
		}
		got.stmt.Release()
	}
	pending.mu.Lock()
	refs := pending.refs
	pending.mu.Unlock()
	if refs != 0 {
		t.Fatalf("preparation left %d active references", refs)
	}
}

type prepareResult struct {
	stmt *Stmt
	err  error
}

func waitForPreparation(t *testing.T, ctx context.Context, done <-chan prepareResult) prepareResult {
	t.Helper()
	select {
	case got := <-done:
		return got
	case <-ctx.Done():
		t.Fatal("eviction deadlocked statement preparation")
		return prepareResult{}
	}
}
