package gorm

import (
	"errors"
	"strconv"

	"database/sql"
)

type Orm struct {
	TableName  string
	PrimaryKey string
	Error      error

	db          *sql.DB
	whereClause []interface{}
	selectStr   string
	orderStr    string
	offsetInt   int
	limitInt    int
}

func (s *Orm) Where(querystring interface{}, args ...interface{}) *Orm {
	s.whereClause = append(s.whereClause, map[string]interface{}{"query": querystring, "args": args})
	return s
}

func (s *Orm) Limit(value interface{}) *Orm {
	switch value := value.(type) {
	case string:
		s.limitInt, _ = strconv.Atoi(value)
	case int:
		s.limitInt = value
	default:
		s.Error = errors.New("Can' understand the value of Limit, Should be int")
	}
	return s
}

func (s *Orm) Offset(value interface{}) *Orm {
	switch value := value.(type) {
	case string:
		s.offsetInt, _ = strconv.Atoi(value)
	case int:
		s.offsetInt = value
	default:
		s.Error = errors.New("Can' understand the value of Offset, Should be int")
	}
	return s
}

func (s *Orm) Order(value interface{}) *Orm {
	switch value := value.(type) {
	case string:
		s.orderStr = value
	default:
		s.Error = errors.New("Can' understand the value of Order, Should be string")
	}
	return s
}

func (s *Orm) Count() int64 {
	return 0
}

func (s *Orm) Select(value interface{}) *Orm {
	switch value := value.(type) {
	case string:
		s.selectStr = value
	default:
		s.Error = errors.New("Can' understand the value of Select, Should be string")
	}
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

func (s *Orm) First(out interface{}) *Orm {
	return s
}

func (s *Orm) Find(out interface{}) *Orm {
	return s
}

func (s *Orm) Or(querystring interface{}, args ...interface{}) *Orm {
	return s
}

func (s *Orm) Not(querystring interface{}, args ...interface{}) *Orm {
	return s
}
