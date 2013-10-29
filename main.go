package gorm

import "database/sql"

type DB struct {
	db     *sql.DB
	driver string
}

func Open(driver, source string) (db DB, err error) {
	db.db, err = sql.Open(driver, source)
	db.driver = driver
	return
}

func (s *DB) SetPool(n int) {
	s.db.SetMaxIdleConns(n)
}

func (s *DB) buildChain() *Chain {
	return &Chain{db: s.db, driver: s.driver}
}

func (s *DB) Where(querystring interface{}, args ...interface{}) *Chain {
	return s.buildChain().Where(querystring, args...)
}

func (s *DB) First(out interface{}, where ...interface{}) *Chain {
	return s.buildChain().First(out, where...)
}

func (s *DB) FirstOrInit(out interface{}, where ...interface{}) *Chain {
	return s.buildChain().FirstOrInit(out, where...)
}

func (s *DB) FirstOrCreate(out interface{}, where ...interface{}) *Chain {
	return s.buildChain().FirstOrCreate(out, where...)
}

func (s *DB) Find(out interface{}, where ...interface{}) *Chain {
	return s.buildChain().Find(out, where...)
}

func (s *DB) Limit(value interface{}) *Chain {
	return s.buildChain().Limit(value)
}

func (s *DB) Offset(value interface{}) *Chain {
	return s.buildChain().Offset(value)
}

func (s *DB) Order(value string, reorder ...bool) *Chain {
	return s.buildChain().Order(value, reorder...)
}

func (s *DB) Select(value interface{}) *Chain {
	return s.buildChain().Select(value)
}

func (s *DB) Save(value interface{}) *Chain {
	return s.buildChain().Save(value)
}

func (s *DB) Delete(value interface{}) *Chain {
	return s.buildChain().Delete(value)
}

func (s *DB) Unscoped() *Chain {
	return s.buildChain().Unscoped()
}

func (s *DB) Exec(sql string) *Chain {
	return s.buildChain().Exec(sql)
}

func (s *DB) Model(value interface{}) *Chain {
	return s.buildChain().Model(value)
}

func (s *DB) Table(name string) *Chain {
	return s.buildChain().Table(name)
}

func (s *DB) CreateTable(value interface{}) *Chain {
	return s.buildChain().CreateTable(value)
}
