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

func (s *DB) buildORM() *Chain {
	return &Chain{db: s.db, driver: s.driver}
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

func (s *DB) Table(name string) *Chain {
	return s.buildORM().Table(name)
}

func (s *DB) CreateTable(value interface{}) *Chain {
	return s.buildORM().CreateTable(value)
}
