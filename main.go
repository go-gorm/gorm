package gorm

import "database/sql"

type DB struct {
	Db *sql.DB
}

func Open(driver, source string) (db *DB, err error) {
	db.Db, err = sql.Open(driver, source)
	return
}

func (s *DB) buildORM() (orm *Orm, err error) {
	orm.Db = s.Db
	return
}

func (s *DB) Where(querystring interface{}, args ...interface{}) (orm *Orm, err error) {
	orm, err = s.buildORM()
	orm.Where(querystring, args)
	return
}

func (s *DB) First() (orm *Orm, err error) {
	orm, err = s.buildORM()
	return
}

func (s *DB) Find() (orm *Orm, err error) {
	orm, err = s.buildORM()
	return
}

func (s *DB) Limit() (orm *Orm, err error) {
	orm, err = s.buildORM()
	return
}

func (s *DB) Offset() (orm *Orm, err error) {
	orm, err = s.buildORM()
	return
}

func (s *DB) Order() (orm *Orm, err error) {
	orm, err = s.buildORM()
	return
}

func (s *DB) Or() (orm *Orm, err error) {
	orm, err = s.buildORM()
	return
}

func (s *DB) Not() (orm *Orm, err error) {
	orm, err = s.buildORM()
	return
}

func (s *DB) Count() (orm *Orm, err error) {
	orm, err = s.buildORM()
	return
}

func (s *DB) Select() (orm *Orm, err error) {
	orm, err = s.buildORM()
	return
}

func (s *DB) Save() (orm *Orm, err error) {
	orm, err = s.buildORM()
	return
}

func (s *DB) Delete() (orm *Orm, err error) {
	orm, err = s.buildORM()
	return
}

func (s *DB) Exec() (orm *Orm, err error) {
	orm, err = s.buildORM()
	return
}
