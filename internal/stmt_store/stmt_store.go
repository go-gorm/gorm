package stmt_store

import (
	"context"
	"database/sql"
	"math"
	"sync"
	"time"

	"gorm.io/gorm/internal/lru"
)

type Stmt struct {
	*sql.Stmt
	Transaction bool
	prepared    chan struct{}
	prepareErr  error
}

func (stmt *Stmt) Error() error {
	return stmt.prepareErr
}

func (stmt *Stmt) Close() error {
	<-stmt.prepared

	if stmt.Stmt != nil {
		return stmt.Stmt.Close()
	}
	return nil
}

// Store defines an interface for managing the caching operations of SQL statements (Stmt).
// This interface provides methods for creating new statements, retrieving all cache keys,
// getting cached statements, setting cached statements, and deleting cached statements.
type Store interface {
	// New creates a new Stmt object and caches it.
	// Parameters:
	//   ctx: The context for the request, which can carry deadlines, cancellation signals, etc.
	//   key: The key representing the SQL query, used for caching and preparing the statement.
	//   isTransaction: Indicates whether this operation is part of a transaction, which may affect the caching strategy.
	//   connPool: A connection pool that provides database connections.
	//   locker: A synchronization lock that is unlocked after initialization to avoid deadlocks.
	// Returns:
	//   *Stmt: A newly created statement object for executing SQL operations.
	//   error: An error if the statement preparation fails.
	New(ctx context.Context, key string, isTransaction bool, connPool ConnPool, locker sync.Locker) (*Stmt, error)

	// Keys returns a slice of all cache keys in the store.
	Keys() []string

	// Get retrieves a Stmt object from the store based on the given key.
	// Parameters:
	//   key: The key used to look up the Stmt object.
	// Returns:
	//   *Stmt: The found Stmt object, or nil if not found.
	//   bool: Indicates whether the corresponding Stmt object was successfully found.
	Get(key string) (*Stmt, bool)

	// Set stores the given Stmt object in the store and associates it with the specified key.
	// Parameters:
	//   key: The key used to associate the Stmt object.
	//   value: The Stmt object to be stored.
	Set(key string, value *Stmt)

	// Delete removes the Stmt object corresponding to the specified key from the store.
	// Parameters:
	//   key: The key associated with the Stmt object to be deleted.
	Delete(key string)
}

// defaultMaxSize defines the default maximum capacity of the cache.
// Its value is the maximum value of the int64 type, which means that when the cache size is not specified,
// the cache can theoretically store as many elements as possible.
// (1 << 63) - 1 is the maximum value that an int64 type can represent.
const (
	defaultMaxSize = math.MaxInt
	// defaultTTL defines the default time-to-live (TTL) for each cache entry.
	// When the TTL for cache entries is not specified, each cache entry will expire after 24 hours.
	defaultTTL = time.Hour * 24
)

// New creates and returns a new Store instance.
//
// Parameters:
//   - size: The maximum capacity of the cache. If the provided size is less than or equal to 0,
//     it defaults to defaultMaxSize.
//   - ttl: The time-to-live duration for each cache entry. If the provided ttl is less than or equal to 0,
//     it defaults to defaultTTL.
//
// This function defines an onEvicted callback that is invoked when a cache entry is evicted.
// The callback ensures that if the evicted value (v) is not nil, its Close method is called asynchronously
// to release associated resources.
//
// Returns:
//   - A Store instance implemented by lruStore, which internally uses an LRU cache with the specified size,
//     eviction callback, and TTL.
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

type lruStore struct {
	lru *lru.LRU[string, *Stmt]
}

func (s *lruStore) Keys() []string {
	return s.lru.Keys()
}

func (s *lruStore) Get(key string) (*Stmt, bool) {
	stmt, ok := s.lru.Get(key)
	if ok && stmt != nil {
		<-stmt.prepared
	}
	return stmt, ok
}

func (s *lruStore) Set(key string, value *Stmt) {
	s.lru.Add(key, value)
}

func (s *lruStore) Delete(key string) {
	s.lru.Remove(key)
}

type ConnPool interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

// New creates a new Stmt object for executing SQL queries.
// It caches the Stmt object for future use and handles preparation and error states.
// Parameters:
//
//	ctx: Context for the request, used to carry deadlines, cancellation signals, etc.
//	key: The key representing the SQL query, used for caching and preparing the statement.
//	isTransaction: Indicates whether this operation is part of a transaction, affecting cache strategy.
//	conn: A connection pool that provides database connections.
//	locker: A synchronization lock that is unlocked after initialization to avoid deadlocks.
//
// Returns:
//
//	*Stmt: A newly created statement object for executing SQL operations.
//	error: An error if the statement preparation fails.
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
