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
	orm := &Orm{db: s.Db, driver: s.Driver}
	return orm
}

func (s *DB) Where(querystring interface{}, args ...interface{}) (orm *Orm) {
	orm = s.buildORM()
	orm.Where(querystring, args...)
	return
}

func (s *DB) First(out interface{}) (orm *Orm) {
	orm = s.buildORM()
	orm.First(out)
	return
}

func (s *DB) Find(out interface{}) (orm *Orm) {
	orm = s.buildORM()
	orm.Find(out)
	return
}

func (s *DB) Limit(value interface{}) (orm *Orm) {
	orm = s.buildORM()
	orm.Limit(value)
	return
}

func (s *DB) Offset(value interface{}) (orm *Orm) {
	orm = s.buildORM()
	orm.Offset(value)
	return
}

func (s *DB) Order(value string, reorder ...bool) (orm *Orm) {
	orm = s.buildORM()
	orm.Order(value, reorder...)
	return
}

func (s *DB) Select(value interface{}) (orm *Orm) {
	orm = s.buildORM()
	orm.Select(value)
	return
}

func (s *DB) Save(value interface{}) (orm *Orm) {
	orm = s.buildORM()
	orm.Save(value)
	return
}

func (s *DB) Delete(value interface{}) (orm *Orm) {
	orm = s.buildORM()
	orm.Delete(value)
	return
}

func (s *DB) Exec(sql string) (orm *Orm) {
	orm = s.buildORM()
	orm.Exec(sql)
	return
}

func (s *DB) CreateTable(value interface{}) (orm *Orm) {
	orm = s.buildORM()
	orm.CreateTable(value)
	return
}
