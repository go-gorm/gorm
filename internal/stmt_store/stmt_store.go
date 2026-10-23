package stmt_store

import (
	"context"
	"database/sql"
	"math"
	"sync"
	"time"

	"gorm.io/gorm/internal/lru"
)

// Stmt wraps a prepared SQL statement with transaction provenance and a
// preparation-completion signal. Obtain it through Store.New/Get; a zero Stmt has no
// completion channel and is not a usable prepared entry.
type Stmt struct {
	*sql.Stmt
	// Transaction records whether preparation used a transaction. The outer GORM cache
	// rejects transaction-only entries for nontransactional execution.
	Transaction bool
	// prepared closes after preparation publishes its result or error. Readers must wait
	// before inspecting the embedded statement or prepareErr.
	prepared chan struct{}
	// prepareErr records a failed preparation and is published by closing prepared.
	prepareErr error
}

// Error returns the recorded preparation failure. It does not wait for preparation;
// call it after Store.Get or another completion synchronization.
func (stmt *Stmt) Error() error {
	return stmt.prepareErr
}

// Close waits for preparation to complete and closes the SQL statement if present. A
// zero-value entry has no completion signal and must not be closed through this method.
func (stmt *Stmt) Close() error {
	<-stmt.prepared

	if stmt.Stmt != nil {
		return stmt.Stmt.Close()
	}
	return nil
}

// Store caches SQL-text-keyed prepared entries. Get synchronizes with preparation;
// Delete triggers resource cleanup. The New lock-transfer contract is essential to
// avoid holding an outer cache lock during a potentially blocking PrepareContext call.
type Store interface {
	// New inserts an in-progress entry, then unlocks the supplied already-held locker
	// before preparing through connPool. The caller must not unlock it again. Completion
	// publishes either the prepared statement or an error; failed entries are removed.
	// Check the returned error before using the entry.
	New(ctx context.Context, key string, isTransaction bool, connPool ConnPool, locker sync.Locker) (*Stmt, error)

	// Keys returns keys of unexpired entries from oldest to most recently used. It does not
	// remove expired entries waiting for cleanup.
	Keys() []string

	// Get returns an unexpired entry and waits for its preparation to finish. Inspect
	// Stmt.Error afterward; an entry found during failed preparation may still report that
	// failure.
	Get(key string) (*Stmt, bool)

	// Set inserts or replaces an entry and renews its cache expiry. Replacing a key does
	// not itself close the previous value; callers own that replacement responsibility.
	Set(key string, value *Stmt)

	// Delete removes the entry if present and schedules its statement closure
	// asynchronously.
	Delete(key string)
}

const (
	// defaultMaxSize is math.MaxInt, used when no positive capacity is provided. It is a
	// practical unbounded default rather than an application memory limit.
	defaultMaxSize = math.MaxInt
	// defaultTTL expires entries 24 hours after insertion or replacement when no positive
	// TTL is configured.
	defaultTTL = time.Hour * 24
)

// New creates an LRU statement store with capacity and TTL. Non-positive size uses
// math.MaxInt and non-positive ttl uses 24 hours. Eviction schedules Stmt.Close in a
// goroutine so resource closure does not block the cache lock. Positive TTLs use a
// background cleanup worker without an exposed stop API.
func New(size int, ttl time.Duration) Store {
	if size <= 0 {
		size = defaultMaxSize
	}

	if ttl <= 0 {
		ttl = defaultTTL
	}

	onEvicted := func(k string, v *Stmt) {
		if v != nil {
			go v.Close()
		}
	}
	return &lruStore{lru: lru.NewLRU[string, *Stmt](size, onEvicted, ttl)}
}

// lruStore implements Store using a synchronized expiring LRU and
// preparation-completion channels.
type lruStore struct {
	// lru stores entries by SQL template and invokes asynchronous closure when entries are
	// removed.
	lru *lru.LRU[string, *Stmt]
}

// Keys returns unexpired SQL keys in oldest-to-newest recency order. Expired entries
// may still await physical cleanup.
func (s *lruStore) Keys() []string {
	return s.lru.Keys()
}

// Get looks up an unexpired entry, updates recency, and waits on its preparation
// channel before returning it. The wait itself has no context cancellation.
func (s *lruStore) Get(key string) (*Stmt, bool) {
	stmt, ok := s.lru.Get(key)
	if ok && stmt != nil {
		<-stmt.prepared
	}
	return stmt, ok
}

// Set inserts or replaces an entry and renews its TTL. Replacing an existing key does
// not invoke the eviction callback for the old value.
func (s *lruStore) Set(key string, value *Stmt) {
	s.lru.Add(key, value)
}

// Delete removes key and schedules Stmt.Close through the LRU eviction callback; it
// does not wait for the close to finish.
func (s *lruStore) Delete(key string) {
	s.lru.Remove(key)
}

// ConnPool is the preparation-only connection capability needed by the statement store.
// It can be implemented by a pool, connection, or transaction.
type ConnPool interface {
	// PrepareContext prepares a SQL template under ctx and returns the owned statement or
	// an error.
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

// New publishes an in-progress entry, releases the supplied held locker, and prepares
// outside that lock. A deferred channel close releases waiting Get/Close calls. On
// failure it records the error, removes the entry, and returns an unusable empty entry
// plus the error; callers must check err first.
func (s *lruStore) New(ctx context.Context, key string, isTransaction bool, conn ConnPool, locker sync.Locker) (_ *Stmt, err error) {
	// Create a Stmt object and set its Transaction property.
	// The prepared channel is used to synchronize the statement preparation state.
	cacheStmt := &Stmt{
		Transaction: isTransaction,
		prepared:    make(chan struct{}),
	}
	// Cache the Stmt object with the associated key.
	s.Set(key, cacheStmt)
	// Unlock after completing initialization to prevent deadlocks.
	locker.Unlock()

	// Ensure the prepared channel is closed after the function execution completes.
	defer close(cacheStmt.prepared)

	// Prepare the SQL statement using the provided connection.
	cacheStmt.Stmt, err = conn.PrepareContext(ctx, key)
	if err != nil {
		// If statement preparation fails, record the error and remove the invalid Stmt object from the cache.
		cacheStmt.prepareErr = err
		s.Delete(key)
		return &Stmt{}, err
	}

	// Return the successfully prepared Stmt object.
	return cacheStmt, nil
}
