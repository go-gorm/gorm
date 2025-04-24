package gorm

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"gorm.io/gorm/internal/lru"
	"reflect"
	"sync"
	"time"
)

type Stmt struct {
	*sql.Stmt
	Transaction bool
	prepared    chan struct{}
	prepareErr  error
}

type PreparedStmtDB struct {
	Stmts StmtStore
	Mux   *sync.RWMutex
	ConnPool
}

const DEFAULT_MAX_SIZE = (1 << 63) - 1
const DEFAULT_TTL = time.Hour * 24

// newPrepareStmtCache creates a new statement cache with the specified maximum size and time-to-live (TTL).
// Parameters:
//   - PrepareStmtMaxSize: An integer specifying the maximum number of prepared statements to cache.
//     If this value is less than or equal to 0, the function will panic.
//   - PrepareStmtTTL: A time.Duration specifying the TTL for cached statements.
//     If this value differs from the default TTL, it will be used instead.
//
// Returns:
//   - A pointer to a store.StmtStore instance configured with the provided parameters.
//
// The function initializes an LRU (Least Recently Used) cache for prepared statements,
// using either the provided size and TTL or default values
func newPrepareStmtCache(PrepareStmtMaxSize int,
	PrepareStmtTTL time.Duration) *StmtStore {
	var lru_size = DEFAULT_MAX_SIZE
	var lru_ttl = DEFAULT_TTL
	var stmts StmtStore
	if PrepareStmtMaxSize < 0 {
		panic("PrepareStmtMaxSize must > 0")
	}
	if PrepareStmtMaxSize != 0 {
		lru_size = PrepareStmtMaxSize
	}
	if PrepareStmtTTL != DEFAULT_TTL {
		lru_ttl = PrepareStmtTTL
	}
	lru := &LruStmtStore{}
	lru.newLru(lru_size, lru_ttl)
	stmts = lru
	return &stmts
}
func NewPreparedStmtDB(connPool ConnPool, PrepareStmtMaxSize int,
	PrepareStmtTTL time.Duration) *PreparedStmtDB {
	return &PreparedStmtDB{
		ConnPool: connPool,
		Stmts: *newPrepareStmtCache(PrepareStmtMaxSize,
			PrepareStmtTTL),
		Mux: &sync.RWMutex{},
	}
}

func (db *PreparedStmtDB) GetDBConn() (*sql.DB, error) {
	if sqldb, ok := db.ConnPool.(*sql.DB); ok {
		return sqldb, nil
	}

	if dbConnector, ok := db.ConnPool.(GetDBConnector); ok && dbConnector != nil {
		return dbConnector.GetDBConn()
	}

	return nil, ErrInvalidDB
}

func (db *PreparedStmtDB) Close() {
	db.Mux.Lock()
	defer db.Mux.Unlock()
	if db.Stmts == nil {
		return
	}

	for _, stmt := range db.Stmts.AllMap() {
		go func(s *Stmt) {
			// make sure the stmt must finish preparation first
			<-s.prepared
			if s.Stmt != nil {
				_ = s.Close()
			}
		}(stmt)
	}
	// setting db.Stmts to nil to avoid further using
	db.Stmts = nil
}

func (sdb *PreparedStmtDB) Reset() {
	sdb.Mux.Lock()
	defer sdb.Mux.Unlock()
	if sdb.Stmts == nil {
		return
	}
	for _, stmt := range sdb.Stmts.AllMap() {
		go func(s *Stmt) {
			// make sure the stmt must finish preparation first
			<-s.prepared
			if s.Stmt != nil {
				_ = s.Close()
			}
		}(stmt)
	}
	//Migrator
	defaultStmt := newPrepareStmtCache(0, 0)
	sdb.Stmts = *defaultStmt
}

func (db *PreparedStmtDB) prepare(ctx context.Context, conn ConnPool, isTransaction bool, query string) (Stmt, error) {
	db.Mux.RLock()
	if db.Stmts != nil {
		if stmt, ok := db.Stmts.get(query); ok && (!stmt.Transaction || isTransaction) {
			db.Mux.RUnlock()
			// wait for other goroutines prepared
			<-stmt.prepared
			if stmt.prepareErr != nil {
				return Stmt{}, stmt.prepareErr
			}

			return *stmt, nil
		}
	}
	db.Mux.RUnlock()

	db.Mux.Lock()
	if db.Stmts != nil {
		// double check
		if stmt, ok := db.Stmts.get(query); ok && (!stmt.Transaction || isTransaction) {
			db.Mux.Unlock()
			// wait for other goroutines prepared
			<-stmt.prepared
			if stmt.prepareErr != nil {
				return Stmt{}, stmt.prepareErr
			}

			return *stmt, nil
		}
	}
	// check db.Stmts first to avoid Segmentation Fault(setting value to nil map)
	// which cause by calling Close and executing SQL concurrently
	if db.Stmts == nil {
		db.Mux.Unlock()
		return Stmt{}, ErrInvalidDB
	}
	// cache preparing stmt first
	cacheStmt := Stmt{Transaction: isTransaction, prepared: make(chan struct{})}
	db.Stmts.set(query, &cacheStmt)
	db.Mux.Unlock()

	// prepare completed
	defer close(cacheStmt.prepared)

	// Reason why cannot lock conn.PrepareContext
	// suppose the maxopen is 1, g1 is creating record and g2 is querying record.
	// 1. g1 begin tx, g1 is requeue because of waiting for the system call, now `db.ConnPool` db.numOpen == 1.
	// 2. g2 select lock `conn.PrepareContext(ctx, query)`, now db.numOpen == db.maxOpen , wait for release.
	// 3. g1 tx exec insert, wait for unlock `conn.PrepareContext(ctx, query)` to finish tx and release.
	stmt, err := conn.PrepareContext(ctx, query)
	if err != nil {
		cacheStmt.prepareErr = err
		db.Mux.Lock()
		db.Stmts.delete(query)
		//delete(db.Stmts.AllMap(), query)
		db.Mux.Unlock()
		return Stmt{}, err
	}

	db.Mux.Lock()
	cacheStmt.Stmt = stmt
	db.Mux.Unlock()

	return cacheStmt, nil
}

func (db *PreparedStmtDB) BeginTx(ctx context.Context, opt *sql.TxOptions) (ConnPool, error) {
	if beginner, ok := db.ConnPool.(TxBeginner); ok {
		tx, err := beginner.BeginTx(ctx, opt)
		return &PreparedStmtTX{PreparedStmtDB: db, Tx: tx}, err
	}

	beginner, ok := db.ConnPool.(ConnPoolBeginner)
	if !ok {
		return nil, ErrInvalidTransaction
	}

	connPool, err := beginner.BeginTx(ctx, opt)
	if err != nil {
		return nil, err
	}
	if tx, ok := connPool.(Tx); ok {
		return &PreparedStmtTX{PreparedStmtDB: db, Tx: tx}, nil
	}
	return nil, ErrInvalidTransaction
}

func (db *PreparedStmtDB) ExecContext(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error) {
	stmt, err := db.prepare(ctx, db.ConnPool, false, query)
	if err == nil {
		result, err = stmt.ExecContext(ctx, args...)
		if errors.Is(err, driver.ErrBadConn) {
			db.Mux.Lock()
			defer db.Mux.Unlock()
			go stmt.Close()
			db.Stmts.delete(query)
			//delete(db.Stmts.AllMap(), query)
		}
	}
	return result, err
}

func (db *PreparedStmtDB) QueryContext(ctx context.Context, query string, args ...interface{}) (rows *sql.Rows, err error) {
	stmt, err := db.prepare(ctx, db.ConnPool, false, query)
	if err == nil {
		rows, err = stmt.QueryContext(ctx, args...)
		if errors.Is(err, driver.ErrBadConn) {
			db.Mux.Lock()
			defer db.Mux.Unlock()

			go stmt.Close()
			db.Stmts.delete(query)
			//delete(db.Stmts.AllMap(), query)
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

func (db *PreparedStmtDB) Ping() error {
	conn, err := db.GetDBConn()
	if err != nil {
		return err
	}
	return conn.Ping()
}

type PreparedStmtTX struct {
	Tx
	PreparedStmtDB *PreparedStmtDB
}

func (db *PreparedStmtTX) GetDBConn() (*sql.DB, error) {
	return db.PreparedStmtDB.GetDBConn()
}

func (tx *PreparedStmtTX) Commit() error {
	if tx.Tx != nil && !reflect.ValueOf(tx.Tx).IsNil() {
		return tx.Tx.Commit()
	}
	return ErrInvalidTransaction
}

func (tx *PreparedStmtTX) Rollback() error {
	if tx.Tx != nil && !reflect.ValueOf(tx.Tx).IsNil() {
		return tx.Tx.Rollback()
	}
	return ErrInvalidTransaction
}

func (tx *PreparedStmtTX) ExecContext(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error) {
	stmt, err := tx.PreparedStmtDB.prepare(ctx, tx.Tx, true, query)
	if err == nil {
		result, err = tx.Tx.StmtContext(ctx, stmt.Stmt).ExecContext(ctx, args...)
		if errors.Is(err, driver.ErrBadConn) {
			tx.PreparedStmtDB.Mux.Lock()
			defer tx.PreparedStmtDB.Mux.Unlock()

			go stmt.Close()
			tx.PreparedStmtDB.Stmts.delete(query)
			//delete(tx.PreparedStmtDB.Stmts.AllMap(), query)
		}
	}
	return result, err
}

func (tx *PreparedStmtTX) QueryContext(ctx context.Context, query string, args ...interface{}) (rows *sql.Rows, err error) {
	stmt, err := tx.PreparedStmtDB.prepare(ctx, tx.Tx, true, query)
	if err == nil {
		rows, err = tx.Tx.StmtContext(ctx, stmt.Stmt).QueryContext(ctx, args...)
		if errors.Is(err, driver.ErrBadConn) {
			tx.PreparedStmtDB.Mux.Lock()
			defer tx.PreparedStmtDB.Mux.Unlock()

			go stmt.Close()
			tx.PreparedStmtDB.Stmts.delete(query)
			//delete(tx.PreparedStmtDB.Stmts.AllMap(), query)
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

func (tx *PreparedStmtTX) Ping() error {
	conn, err := tx.GetDBConn()
	if err != nil {
		return err
	}
	return conn.Ping()
}

type StmtStore interface {
	get(key string) (*Stmt, bool)
	set(key string, value *Stmt)
	delete(key string)
	AllMap() map[string]*Stmt
}

/*
	type DefaultStmtStore struct {
		defaultStmt map[string]*Stmt
	}

	func (s *DefaultStmtStore) Init() *DefaultStmtStore {
		s.defaultStmt = make(map[string]*Stmt)
		return s
	}

	func (s *DefaultStmtStore) AllMap() map[string]*Stmt {
		return s.defaultStmt
	}

	func (s *DefaultStmtStore) Get(key string) (*Stmt, bool) {
		stmt, ok := s.defaultStmt[key]
		return stmt, ok
	}

	func (s *DefaultStmtStore) Set(key string, value *Stmt) {
		s.defaultStmt[key] = value
	}

	func (s *DefaultStmtStore) Delete(key string) {
		delete(s.defaultStmt, key)
	}
*/
type LruStmtStore struct {
	lru *lru.LRU[string, *Stmt]
}

func (s *LruStmtStore) newLru(size int, ttl time.Duration) {
	onEvicted := func(k string, v *Stmt) {
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
	s.lru = lru.NewLRU[string, *Stmt](size, onEvicted, ttl)
}

func (s *LruStmtStore) AllMap() map[string]*Stmt {
	return s.lru.KeyValues()
}
func (s *LruStmtStore) get(key string) (*Stmt, bool) {
	stmt, ok := s.lru.Get(key)
	return stmt, ok
}

func (s *LruStmtStore) set(key string, value *Stmt) {
	s.lru.Add(key, value)
}

func (s *LruStmtStore) delete(key string) {
	s.lru.Remove(key)
}
