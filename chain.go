package gorm

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"

	"strconv"
)

type Chain struct {
	db     *sql.DB
	driver string
	value  interface{}

	Errors []error
	Error  error

	whereClause []map[string]interface{}
	orClause    []map[string]interface{}
	selectStr   string
	orderStrs   []string
	offsetStr   string
	limitStr    string
}

func (s *Chain) err(err error) {
	if err != nil {
		s.Errors = append(s.Errors, err)
		s.Error = err
	}
}

func (s *Chain) do(value interface{}) *Do {
	var do Do
	do.chain = s
	do.db = s.db
	do.driver = s.driver

	do.whereClause = s.whereClause
	do.orClause = s.orClause
	do.selectStr = s.selectStr
	do.orderStrs = s.orderStrs
	do.offsetStr = s.offsetStr
	do.limitStr = s.limitStr

	do.setModel(value)
	return &do
}

func (s *Chain) Model(model interface{}) *Chain {
	s.value = model
	return s
}

func (s *Chain) Where(querystring interface{}, args ...interface{}) *Chain {
	s.whereClause = append(s.whereClause, map[string]interface{}{"query": querystring, "args": args})
	return s
}

func (s *Chain) Limit(value interface{}) *Chain {
	switch value := value.(type) {
	case string:
		s.limitStr = value
	case int:
		if value < 0 {
			s.limitStr = ""
		} else {
			s.limitStr = strconv.Itoa(value)
		}
	default:
		s.err(errors.New("Can' understand the value of Limit, Should be int"))
	}
	return s
}

func (s *Chain) Offset(value interface{}) *Chain {
	switch value := value.(type) {
	case string:
		s.offsetStr = value
	case int:
		if value < 0 {
			s.offsetStr = ""
		} else {
			s.offsetStr = strconv.Itoa(value)
		}
	default:
		s.err(errors.New("Can' understand the value of Offset, Should be int"))
	}
	return s
}

func (s *Chain) Order(value string, reorder ...bool) *Chain {
	defer s.validSql(value)
	if len(reorder) > 0 && reorder[0] {
		s.orderStrs = append([]string{}, value)
	} else {
		s.orderStrs = append(s.orderStrs, value)
	}
	return s
}

func (s *Chain) Count() int64 {
	return 0
}

func (s *Chain) Select(value interface{}) *Chain {
	defer func() { s.validSql(s.selectStr) }()

	switch value := value.(type) {
	case string:
		s.selectStr = value
	default:
		s.err(errors.New("Can' understand the value of Select, Should be string"))
	}

	return s
}

func (s *Chain) Save(value interface{}) *Chain {
	s.do(value).save()
	return s
}

func (s *Chain) Delete(value interface{}) *Chain {
	s.do(value).delete()
	return s
}

func (s *Chain) Update(column string, value string) *Chain {
	return s
}

func (s *Chain) Updates(values map[string]string) *Chain {
	return s
}

func (s *Chain) Exec(sql string) *Chain {
	var err error
	_, err = s.db.Exec(sql)
	s.err(err)
	return s
}

func (s *Chain) First(out interface{}) *Chain {
	s.do(out).query()
	return s
}

func (s *Chain) Find(out interface{}) *Chain {
	s.do(out).query()
	return s
}

func (s *Chain) Pluck(column string, value interface{}) (orm *Chain) {
	s.Select(column).do(s.value).pluck(value)
	return s
}

func (s *Chain) Or(querystring interface{}, args ...interface{}) *Chain {
	s.orClause = append(s.orClause, map[string]interface{}{"query": querystring, "args": args})
	return s
}

func (s *Chain) CreateTable(value interface{}) *Chain {
	s.do(value).createTable().Exec()
	return s
}

func (s *Chain) validSql(str string) (result bool) {
	result = regexp.MustCompile("^\\s*[\\w][\\w\\s,.]*[\\w]\\s*$").MatchString(str)
	if !result {
		s.err(errors.New(fmt.Sprintf("SQL is not valid, %s", str)))
	}
	return
}
