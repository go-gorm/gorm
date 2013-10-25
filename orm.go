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
	Error      bool
}

func (s *Orm) Where(querystring interface{}, args ...interface{}) *Orm {
	return s
}

func (s *Orm) First(out interface{}) *Orm {
	return s
}

func (s *Orm) Find(out interface{}) *Orm {
	return s
}

func (s *Orm) Limit(value interface{}) *Orm {
	return s
}

func (s *Orm) Offset(value interface{}) *Orm {
	return s
}

func (s *Orm) Order(value interface{}) *Orm {
	return s
}

func (s *Orm) Or(querystring interface{}, args ...interface{}) *Orm {
	return s
}

func (s *Orm) Not(querystring interface{}, args ...interface{}) *Orm {
	return s
}

func (s *Orm) Count() int64 {
	return 0
}

func (s *Orm) Select(querystring string) *Orm {
	return s
}

func (s *Orm) Save(value interface{}) *Orm {
	return s
}

func (s *Orm) Delete(value interface{}) *Orm {
	return s
}

func (s *Orm) Exec(sql string) *Orm {
	return s
}

func (s *Orm) Explain() string {
	return ""
}
