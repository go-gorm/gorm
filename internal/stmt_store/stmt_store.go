package stmt_store

import (
	"context"
	"database/sql"
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

type Store interface {
	New(ctx context.Context, key string, isTransaction bool, connPool ConnPool, locker sync.Locker) (*Stmt, error)
	Keys() []string
	Get(key string) (*Stmt, bool)
	Set(key string, value *Stmt)
	Delete(key string)
}

type LRUStore struct {
	lru *lru.LRU[string, *Stmt]
}

const (
	defaultMaxSize = (1 << 63) - 1
	defaultTTL     = time.Hour * 24
)

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
	return &LRUStore{lru: lru.NewLRU[string, *Stmt](size, onEvicted, ttl)}
}

func (s *LRUStore) Keys() []string {
	return s.lru.Keys()
}

func (s *LRUStore) Get(key string) (*Stmt, bool) {
	stmt, ok := s.lru.Get(key)
	if ok && stmt != nil {
		<-stmt.prepared
	}
	return stmt, ok
}

func (s *LRUStore) Set(key string, value *Stmt) {
	s.lru.Add(key, value)
}

func (s *LRUStore) Delete(key string) {
	s.lru.Remove(key)
}

type ConnPool interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
}

func (s *LRUStore) New(ctx context.Context, key string, isTransaction bool, conn ConnPool, locker sync.Locker) (_ *Stmt, err error) {
	cacheStmt := &Stmt{
		Transaction: isTransaction,
		prepared:    make(chan struct{}),
	}
	s.Set(key, cacheStmt)
	locker.Unlock()

	defer close(cacheStmt.prepared)

	cacheStmt.Stmt, err = conn.PrepareContext(ctx, key)
	if err != nil {
		cacheStmt.prepareErr = err
		s.Delete(key)
		return &Stmt{}, err
	}

	return cacheStmt, nil
}
