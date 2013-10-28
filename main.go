package gorm

import (
	"database/sql"
	_ "github.com/lib/pq"
)

type DB struct {
	Db     *sql.DB
	Driver string
}

func Open(driver, source string) (db DB, err error) {
	db.Db, err = sql.Open(driver, source)
	db.Driver = driver
	// SetMaxIdleConns pools
	return
}

func (s *DB) SetPool(n int) {
	s.Db.SetMaxIdleConns(n)
}

func (s *DB) buildORM() *Chain {
	return &Chain{db: s.Db, driver: s.Driver}
}

func (s *DB) Where(querystring interface{}, args ...interface{}) *Chain {
	return s.buildORM().Where(querystring, args...)
}

func (s *DB) First(out interface{}, where ...interface{}) *Chain {
	return s.buildORM().First(out, where...)
}

func (s *DB) Find(out interface{}, where ...interface{}) *Chain {
	return s.buildORM().Find(out, where...)
}

func (s *DB) Limit(value interface{}) *Chain {
	return s.buildORM().Limit(value)
}

func (s *DB) Offset(value interface{}) *Chain {
	return s.buildORM().Offset(value)
}

func (s *DB) Order(value string, reorder ...bool) *Chain {
	return s.buildORM().Order(value, reorder...)
}

func (s *DB) Select(value interface{}) *Chain {
	return s.buildORM().Select(value)
}

func (s *DB) Save(value interface{}) *Chain {
	return s.buildORM().Save(value)
}

func (s *DB) Delete(value interface{}) *Chain {
	return s.buildORM().Delete(value)
}

func (s *DB) Exec(sql string) *Chain {
	return s.buildORM().Exec(sql)
}

func (s *DB) Model(value interface{}) *Chain {
	return s.buildORM().Model(value)
}

func (s *DB) CreateTable(value interface{}) *Chain {
	return s.buildORM().CreateTable(value)
}
