package gorm

import (
	"errors"
	"fmt"
	"regexp"
)

type Chain struct {
	d          *DB
	db         sql_common
	value      interface{}
	debug_mode bool

	Errors []error
	Error  error

	whereClause        []map[string]interface{}
	orClause           []map[string]interface{}
	notClause          []map[string]interface{}
	initAttrs          []interface{}
	assignAttrs        []interface{}
	selectStr          string
	orderStrs          []string
	offsetStr          string
	limitStr           string
	specifiedTableName string
	unscoped           bool
}

func (s *Chain) driver() string {
	return s.d.driver
}

func (s *Chain) err(err error) error {
	if err != nil {
		s.Errors = append(s.Errors, err)
		s.Error = err
		s.warn(err)
	}
	return err
}

func (s *Chain) hasError() bool {
	return len(s.Errors) > 0
}

func (s *Chain) deleteLastError() {
	s.Error = nil
	s.Errors = s.Errors[:len(s.Errors)-1]
}

func (s *Chain) do(value interface{}) *Do {
	do := Do{
		chain:              s,
		db:                 s.db,
		whereClause:        s.whereClause,
		orClause:           s.orClause,
		notClause:          s.notClause,
		selectStr:          s.selectStr,
		orderStrs:          s.orderStrs,
		offsetStr:          s.offsetStr,
		limitStr:           s.limitStr,
		specifiedTableName: s.specifiedTableName,
		unscoped:           s.unscoped,
	}

	s.value = value
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

func (s *Chain) Not(querystring interface{}, args ...interface{}) *Chain {
	s.notClause = append(s.notClause, map[string]interface{}{"query": querystring, "args": args})
	return s
}

func (s *Chain) Limit(value interface{}) *Chain {
	if str, err := getInterfaceAsString(value); err == nil {
		s.limitStr = str
	} else {
		s.err(errors.New("Can' understand the value of Limit, Should be int"))
	}
	return s
}

func (s *Chain) Offset(value interface{}) *Chain {
	if str, err := getInterfaceAsString(value); err == nil {
		s.offsetStr = str
	} else {
		s.err(errors.New("Can' understand the value of Offset, Should be int"))
	}
	return s
}

func (s *Chain) Order(value string, reorder ...bool) *Chain {
	defer s.validSql(value)
	if len(reorder) > 0 && reorder[0] {
		s.orderStrs = []string{value}
	} else {
		s.orderStrs = append(s.orderStrs, value)
	}
	return s
}

func (s *Chain) Count(value interface{}) *Chain {
	s.Select("count(*)").do(s.value).count(value)
	return s
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
	do := s.do(value).begin()
	do.save()
	do.commit_or_rollback()
	return s
}

func (s *Chain) Delete(value interface{}) *Chain {
	do := s.do(value).begin()
	do.delete()
	do.commit_or_rollback()
	return s
}

func (s *Chain) Update(attrs ...interface{}) *Chain {
	return s.Updates(toSearchableMap(attrs...), true)
}

func (s *Chain) Updates(values interface{}, ignore_protected_attrs ...bool) *Chain {
	do := s.do(s.value).begin().setUpdateAttrs(values, ignore_protected_attrs...)
	do.update()
	do.commit_or_rollback()
	return s
}

func (s *Chain) Exec(sql string) *Chain {
	s.do(nil).exec(sql)
	return s
}

func (s *Chain) First(out interface{}, where ...interface{}) *Chain {
	s.do(out).where(where...).first()
	return s
}

func (s *Chain) Last(out interface{}, where ...interface{}) *Chain {
	s.do(out).where(where...).last()
	return s
}

func (s *Chain) Attrs(attrs ...interface{}) *Chain {
	s.initAttrs = append(s.initAttrs, toSearchableMap(attrs...))
	return s
}

func (s *Chain) Assign(attrs ...interface{}) *Chain {
	s.assignAttrs = append(s.assignAttrs, toSearchableMap(attrs...))
	return s
}

func (s *Chain) FirstOrInit(out interface{}, where ...interface{}) *Chain {
	if s.First(out, where...).Error != nil {
		s.deleteLastError()
		s.do(out).where(where...).where(s.initAttrs).where(s.assignAttrs).initializeWithSearchCondition()
	} else {
		if len(s.assignAttrs) > 0 {
			s.do(out).setUpdateAttrs(s.assignAttrs).prepareUpdateAttrs()
		}
	}
	return s
}

func (s *Chain) FirstOrCreate(out interface{}, where ...interface{}) *Chain {
	if s.First(out, where...).Error != nil {
		s.deleteLastError()
		s.do(out).where(where...).where(s.initAttrs).where(s.assignAttrs).initializeWithSearchCondition()
		s.Save(out)
	} else {
		if len(s.assignAttrs) > 0 {
			s.do(out).setUpdateAttrs(s.assignAttrs).update()
		}
	}
	return s
}

func (s *Chain) Find(out interface{}, where ...interface{}) *Chain {
	s.do(out).where(where...).query()
	return s
}

func (s *Chain) Pluck(column string, value interface{}) (orm *Chain) {
	s.do(s.value).pluck(column, value)
	return s
}

func (s *Chain) Or(querystring interface{}, args ...interface{}) *Chain {
	s.orClause = append(s.orClause, map[string]interface{}{"query": querystring, "args": args})
	return s
}

func (s *Chain) CreateTable(value interface{}) *Chain {
	s.do(value).createTable()
	return s
}

func (s *Chain) DropTable(value interface{}) *Chain {
	s.do(value).dropTable()
	return s
}

func (s *Chain) AutoMigrate(value interface{}) *Chain {
	s.do(value).autoMigrate()
	return s
}

func (s *Chain) Unscoped() *Chain {
	s.unscoped = true
	return s
}

func (s *Chain) Table(name string) *Chain {
	s.specifiedTableName = name
	return s
}

func (s *Chain) Related(value interface{}, foreign_keys ...string) *Chain {
	original_value := s.value
	s.do(value).related(original_value, foreign_keys...)
	return s
}

func (s *Chain) Begin() *Chain {
	if db, ok := s.db.(sql_db); ok {
		tx, err := db.Begin()
		s.db = interface{}(tx).(sql_common)
		s.err(err)
	} else {
		s.err(errors.New("Can't start a transaction."))
	}

	return s
}

func (s *Chain) Debug() *Chain {
	s.debug_mode = true
	return s
}

func (s *Chain) Commit() *Chain {
	if db, ok := s.db.(sql_tx); ok {
		s.err(db.Commit())
	} else {
		s.err(errors.New("Commit is not supported, no database transaction found."))
	}
	return s
}

func (s *Chain) Rollback() *Chain {
	if db, ok := s.db.(sql_tx); ok {
		s.err(db.Rollback())
	} else {
		s.err(errors.New("Rollback is not supported, no database transaction found."))
	}
	return s
}

func (s *Chain) validSql(str string) (result bool) {
	result = regexp.MustCompile("^\\s*[\\w\\s,.*()]*\\s*$").MatchString(str)
	if !result {
		s.err(errors.New(fmt.Sprintf("SQL is not valid, %s", str)))
	}
	return
}
