package gorm

import "database/sql"

type Orm struct {
	TableName  string
	PrimaryKey string
	Error      bool

	db          *sql.DB
	whereClause []interface{}
	orderStr    string
	offsetInt   int64
	limitInt    int64
}

func (s *Orm) Where(querystring interface{}, args ...interface{}) *Orm {
	s.whereClause = append(s.whereClause, map[string]interface{}{"query": querystring, "args": args})
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

func (s *Orm) Update(column string, value string) *Orm {
	return s
}

func (s *Orm) Updates(values map[string]string) *Orm {
	return s
}

func (s *Orm) Exec(sql string) *Orm {
	return s
}

func (s *Orm) Explain() string {
	return ""
}
