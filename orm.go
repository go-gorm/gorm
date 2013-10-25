package gorm

import "database/sql"

type Orm struct {
	Db         *sql.DB
	TableName  string
	WhereStr   string
	OrderStr   string
	PrimaryKey string
	OffsetInt  int64
	LimitInt   int64
}

func (s *Orm) Where(querystring interface{}, args ...interface{}) (*Orm, error) {
	return s, nil
}

func (s *Orm) First() (*Orm, error) {
	return s, nil
}

func (s *Orm) Find() (*Orm, error) {
	return s, nil
}

func (s *Orm) Limit() (*Orm, error) {
	return s, nil
}

func (s *Orm) Offset() (*Orm, error) {
	return s, nil
}

func (s *Orm) Order() (*Orm, error) {
	return s, nil
}

func (s *Orm) Or() (*Orm, error) {
	return s, nil
}

func (s *Orm) Not() (*Orm, error) {
	return s, nil
}

func (s *Orm) Count() (*Orm, error) {
	return s, nil
}

func (s *Orm) Select() (*Orm, error) {
	return s, nil
}

func (s *Orm) Save() (*Orm, error) {
	return s, nil
}

func (s *Orm) Delete() (*Orm, error) {
	return s, nil
}

func (s *Orm) Exec() (*Orm, error) {
	return s, nil
}
