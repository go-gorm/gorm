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

func (s *DB) buildORM() *Orm {
	return &Orm{db: s.Db, driver: s.Driver}
}

func (s *DB) Where(querystring interface{}, args ...interface{}) *Orm {
	return s.buildORM().Where(querystring, args...)
}

func (s *DB) First(out interface{}) *Orm {
	return s.buildORM().First(out)
}

func (s *DB) Find(out interface{}) *Orm {
	return s.buildORM().Find(out)
}

func (s *DB) Limit(value interface{}) *Orm {
	return s.buildORM().Limit(value)
}

func (s *DB) Offset(value interface{}) *Orm {
	return s.buildORM().Offset(value)
}

func (s *DB) Order(value string, reorder ...bool) *Orm {
	return s.buildORM().Order(value, reorder...)
}

func (s *DB) Select(value interface{}) *Orm {
	return s.buildORM().Select(value)
}

func (s *DB) Save(value interface{}) *Orm {
	return s.buildORM().Save(value)
}

func (s *DB) Delete(value interface{}) *Orm {
	return s.buildORM().Delete(value)
}

func (s *DB) Exec(sql string) *Orm {
	return s.buildORM().Exec(sql)
}

func (s *DB) Model(value interface{}) *Orm {
	return s.buildORM().Model(value)
}

func (s *DB) CreateTable(value interface{}) *Orm {
	return s.buildORM().CreateTable(value)
}
