package stmt_store

import (
	"database/sql"
	"fmt"
	"time"

	"gorm.io/gorm/internal/lru"
)

type Stmt struct {
	*sql.Stmt
	Transaction bool
	prepared    chan struct{}
	prepareErr  error
}

func NewStmt(isTransaction bool) *Stmt {
	return &Stmt{
		Transaction: isTransaction,
		prepared:    make(chan struct{}),
	}
}

func (stmt *Stmt) Done() {
	close(stmt.prepared)
}

func (stmt *Stmt) AddError(err error) {
	stmt.prepareErr = err
}

func (stmt *Stmt) Error() error {
	<-stmt.prepared

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
	Get(key string) (*Stmt, bool)
	Set(key string, value *Stmt)
	Delete(key string)
	AllMap() map[string]*Stmt
}

type StmtStore struct {
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
			go func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Print("close stmt err panic ")
					}
				}()
				err := v.Close()
				if err != nil {
					fmt.Print("close stmt err: ", err.Error())
				}
			}()
		}
	}
	return &StmtStore{lru: lru.NewLRU[string, *Stmt](size, onEvicted, ttl)}
}

func (s *StmtStore) AllMap() map[string]*Stmt {
	return s.lru.KeyValues()
}

func (s *StmtStore) Get(key string) (*Stmt, bool) {
	stmt, ok := s.lru.Get(key)
	return stmt, ok
}

func (s *StmtStore) Set(key string, value *Stmt) {
	s.lru.Add(key, value)
}

func (s *StmtStore) Delete(key string) {
	s.lru.Remove(key)
}
