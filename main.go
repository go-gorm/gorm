package gorm

import (
	"database/sql"
	"errors"
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

func (s *DB) LogMode(b bool) *DB {
	s.logMode = b
	return s
}

func (s *DB) SingularTable(b bool) {
	s.parent.singularTable = b
}

func (s *DB) Where(query interface{}, args ...interface{}) *DB {
	return s.clone().search.where(query, args...).db
}

func (s *DB) Or(query interface{}, args ...interface{}) *DB {
	return s.clone().search.or(query, args...).db
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
	return s.clone().do(out).where(where...).first().db
}

func (s *DB) Last(out interface{}, where ...interface{}) *DB {
	return s.clone().do(out).where(where...).last().db
}

func (s *DB) Find(out interface{}, where ...interface{}) *DB {
	return s.clone().do(out).where(where...).query().db
}

func (s *DB) Attrs(attrs ...interface{}) *DB {
	return s.clone().search.attrs(attrs...).db
}

func (s *DB) Assign(attrs ...interface{}) *DB {
	return s.clone().search.assign(attrs...).db
}

func (s *DB) FirstOrInit(out interface{}, where ...interface{}) *DB {
	if s.clone().First(out, where...).Error != nil {
		return s.clone().do(out).where(where).initialize().db
	} else {
		if len(s.search.assignAttrs) > 0 {
			return s.clone().do(out).updateAttrs(s.search.assignAttrs).db
		}
	}
	return s
}

func (s *DB) FirstOrCreate(out interface{}, where ...interface{}) *DB {
	if s.clone().First(out, where...).Error != nil {
		return s.clone().do(out).where(where...).initialize().db.Save(out)
	} else {
		if len(s.search.assignAttrs) > 0 {
			return s.clone().do(out).updateAttrs(s.search.assignAttrs).update().db
		}
	}
	return s
}

func (s *DB) Update(attrs ...interface{}) *DB {
	return s.Updates(toSearchableMap(attrs...), true)
}

func (s *DB) Updates(values interface{}, ignore_protected_attrs ...bool) *DB {
	return s.clone().do(s.data).begin().updateAttrs(values, ignore_protected_attrs...).update().commit_or_rollback().db
}

func (s *DB) Save(value interface{}) *DB {
	return s.clone().do(value).begin().save().commit_or_rollback().db
}

func (s *DB) Delete(value interface{}) *DB {
	return s.clone().do(value).begin().delete().commit_or_rollback().db
}

func (s *DB) Exec(sql string) *DB {
	return s.do(nil).exec(sql).db
}

func (s *DB) Model(value interface{}) *DB {
	c := s.clone()
	c.data = value
	return c
}

func (s *DB) Related(value interface{}, foreign_keys ...string) *DB {
	old_data := s.data
	return s.do(value).related(old_data, foreign_keys...).db
}

func (s *DB) Pluck(column string, value interface{}) *DB {
	return s.do(s.data).pluck(column, value).db
}

func (s *DB) Count(value interface{}) *DB {
	return s.do(s.data).count(value).db
}

func (s *DB) Table(name string) *DB {
	return s.clone().search.table(name).db
}

// Debug
func (s *DB) Debug() *DB {
	return s.clone().LogMode(true)
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
