package gorm

import (
	"errors"

	"database/sql"
)
import "github.com/jinzhu/gorm/dialect"

type DB struct {
	db            sqlCommon
	parent        *DB
	search        *search
	data          interface{}
	Error         error
	dialect       dialect.Dialect
	tagIdentifier string
	singularTable bool
	logger        Logger
	logMode       bool
}

func Open(driver, source string) (db DB, err error) {
	db.db, err = sql.Open(driver, source)
	db.dialect = dialect.New(driver)
	db.parent = &db
	return
}

func (s *DB) SetPool(n int) {
	if db, ok := s.db.(sqlDb); ok {
		db.SetMaxIdleConns(n)
	}
}

func (s *DB) SetTagIdentifier(str string) {
	s.parent.tagIdentifier = str
}

func (s *DB) SetLogger(l Logger) {
	s.parent.logger = l
}

func (s *DB) LogMode(b bool) {
	s.parent.logMode = b
}

func (s *DB) SingularTable(b bool) {
	s.parent.singularTable = b
}

func (s *DB) clone() *DB {
	db := &DB{db: s.db, parent: s.parent, search: s.parent.search.clone()}
	db.search.db = db
	return db
}

func (s *DB) do(data interface{}) *Do {
	s.data = data
	return &Do{db: s}
}

func (s *DB) err(err error) error {
	if err != nil {
		s.Error = err
		s.warn(err)
	}
	return err
}

func (s *DB) hasError() bool {
	return s.Error != nil
}

func (s *DB) Where(query interface{}, args ...interface{}) *DB {
	return s.clone().search.where(query, args...).db
}

func (s *DB) Or(query interface{}, args ...interface{}) *DB {
	return s.clone().search.where(query, args...).db
}

func (s *DB) Not(query interface{}, args ...interface{}) *DB {
	return s.clone().search.not(query, args...).db
}

func (s *DB) Limit(value interface{}) *DB {
	return s.clone().search.limit(value).db
}

func (s *DB) Offset(value interface{}) *DB {
	return s.clone().search.offset(value).db
}

func (s *DB) Order(value string, reorder ...bool) *DB {
	return s.clone().search.order(value, reorder...).db
}

func (s *DB) Select(value interface{}) *DB {
	return s.clone().search.selects(value).db
}

func (s *DB) Unscoped() *DB {
	return s.clone().search.unscoped().db
}

func (s *DB) First(out interface{}, where ...interface{}) *DB {
	s.clone().search.limit(1).where(where[0], where[1:]).db.do(out).first()
	return s
}

func (s *DB) Last(out interface{}, where ...interface{}) *DB {
	s.clone().search.limit(1).where(where[0], where[1:]).db.do(out).last()
	return s
}

func (s *DB) Find(out interface{}, where ...interface{}) *DB {
	s.clone().search.where(where[0], where[1:]).db.do(out).query()
	return s
}

func (s *DB) Attrs(attrs ...interface{}) *DB {
	return s.clone().search.attrs(attrs...).db
}

func (s *DB) Assign(attrs ...interface{}) *DB {
	return s.clone().search.assign(attrs...).db
}

func (s *DB) FirstOrInit(out interface{}, where ...interface{}) *DB {
	if s.First(out, where...).Error != nil {
		s.clone().do(out).where(where).initialize()
	} else {
		if len(s.search.assignAttrs) > 0 {
			s.do(out).updateAttrs(s.assignAttrs) //updated or not
		}
	}
	return s
}

func (s *DB) FirstOrCreate(out interface{}, where ...interface{}) *DB {
	if s.First(out, where...).Error != nil {
		s.clone().do(out).where(where...).initialize()
		s.Save(out)
	} else {
		if len(s.search.assignAttrs) > 0 {
			s.do(out).updateAttrs(s.search.assignAttrs).update()
		}
	}
	return s
}

func (s *DB) Save(value interface{}) *DB {
	s.do(value).begin().save().commit_or_rollback()
	return s
}

func (s *DB) Delete(value interface{}) *DB {
	s.do(value).bengin().delete(value).commit_or_rollback()
	return s
}

func (s *DB) Exec(sql string) *DB {
	s.do(nil).exec(sql)
	return s
}

func (s *DB) Model(value interface{}) *DB {
	c := s.clone()
	c.data = value
	return c
}

func (s *DB) Related(value interface{}, foreign_keys ...string) *DB {
	s.clone().do(value).related(s.value, foreign_keys...)
	return s
}

func (s *DB) Pluck(column string, value interface{}) *DB {
	s.clone().search.selects(column).do(s.value).pluck(column, value)
	return s
}

func (s *DB) Count(value interface{}) *DB {
	s.clone().search.selects("count(*)").do(s.value).count(value)
	return s
}

func (s *DB) Table(name string) *DB {
	return s.clone().table(name).db
}

// Debug
func (s *DB) Debug() *DB {
	s.logMode = true
	return s
}

// Transactions
func (s *DB) Begin() *DB {
	c := s.clone()
	if db, ok := c.db.(sqlDb); ok {
		tx, err := db.Begin()
		c.db = interface{}(tx).(sqlCommon)
		c.err(err)
	} else {
		c.err(errors.New("Can't start a transaction."))
	}
	return c
}

func (s *DB) Commit() *DB {
	if db, ok := s.db.(sqlTx); ok {
		s.err(db.Commit())
	} else {
		s.err(NoValidTransaction)
	}
	return s
}

func (s *DB) Rollback() *DB {
	if db, ok := s.db.(sqlTx); ok {
		s.err(db.Rollback())
	} else {
		s.err(NoValidTransaction)
	}
	return s
}

// Migrations
func (s *DB) CreateTable(value interface{}) *DB {
	s.do(value).createTable()
	return s
}

func (s *DB) DropTable(value interface{}) *DB {
	s.do(value).dropTable()
	return s
}

func (s *DB) AutoMigrate(value interface{}) *DB {
	s.do(value).autoMigrate()
	return s
}

func (s *DB) UpdateColumn(column string, typ string) *DB {
	s.do(s.data).updateColumn(column, typ)
	return s
}

func (s *DB) DropColumn(column string) *DB {
	s.do(s.data).dropColumn(column)
	return s
}

func (s *DB) AddIndex(column string, index_name ...string) *DB {
	s.do(s.data).addIndex(column, index_name...)
	return s
}

func (s *DB) RemoveIndex(column string) *DB {
	s.do(s.data).removeIndex(column)
	return s
}
