package store

import (
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/internal/lru"
	"time"
)

type StmtStore interface {
	Get(key string) (*gorm.Stmt, bool)
	Set(key string, value *gorm.Stmt)
	Delete(key string)
	AllMap() map[string]*gorm.Stmt
}

/*
	type DefaultStmtStore struct {
		defaultStmt map[string]*gorm.Stmt
	}

	func (s *DefaultStmtStore) Init() *DefaultStmtStore {
		s.defaultStmt = make(map[string]*gorm.Stmt)
		return s
	}

	func (s *DefaultStmtStore) AllMap() map[string]*gorm.Stmt {
		return s.defaultStmt
	}

	func (s *DefaultStmtStore) Get(key string) (*gorm.Stmt, bool) {
		stmt, ok := s.defaultStmt[key]
		return stmt, ok
	}

	func (s *DefaultStmtStore) Set(key string, value *gorm.Stmt) {
		s.defaultStmt[key] = value
	}

	func (s *DefaultStmtStore) Delete(key string) {
		delete(s.defaultStmt, key)
	}
*/
type LruStmtStore struct {
	lru *lru.LRU[string, *gorm.Stmt]
}

func (s *LruStmtStore) NewLru(size int, ttl time.Duration) {
	onEvicted := func(k string, v *gorm.Stmt) {
		if v != nil {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						fmt.Print("close stmt err panic ")
					}
				}()
				if v != nil {
					err := v.Close()
					if err != nil {
						//
						fmt.Print("close stmt err: ", err.Error())
					}
				}
			}()
		}
	}
	s.lru = lru.NewLRU[string, *gorm.Stmt](size, onEvicted, ttl)
}

func (s *LruStmtStore) AllMap() map[string]*gorm.Stmt {
	return s.lru.KeyValues()
}
func (s *LruStmtStore) Get(key string) (*gorm.Stmt, bool) {
	stmt, ok := s.lru.Get(key)
	return stmt, ok
}

func (s *LruStmtStore) Set(key string, value *gorm.Stmt) {
	s.lru.Add(key, value)
}

func (s *LruStmtStore) Delete(key string) {
	s.lru.Remove(key)
}
