package gorm

import (
	"database/sql"

	"github.com/jinzhu/gorm/dialect"
)

type DB struct {
	Value         interface{}
	callback      *callback
	Error         error
	db            sqlCommon
	parent        *DB
	search        *search
	logMode       int
	logger        logger
	dialect       dialect.Dialect
	tagIdentifier string
	singularTable bool
}

func Open(driver, source string) (DB, error) {
	var err error
	db := DB{dialect: dialect.New(driver), tagIdentifier: "sql", logger: defaultLogger, callback: DefaultCallback}
	db.db, err = sql.Open(driver, source)
	db.parent = &db
	return db, err
}

func (s *DB) DB() *sql.DB {
	return s.db.(*sql.DB)
}

func (s *DB) SetTagIdentifier(str string) {
	s.parent.tagIdentifier = str
}

func (s *DB) SetLogger(l logger) {
	s.parent.logger = l
}

func (s *DB) LogMode(b bool) *DB {
	if b {
		s.logMode = 2
	} else {
		s.logMode = 1
	}
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

func (s *DB) Group(query string) *DB {
	return s.clone().search.group(query).db
}

func (s *DB) Having(query string, values ...interface{}) *DB {
	return s.clone().search.having(query, values...).db
}

func (s *DB) Joins(query string) *DB {
	return s.clone().search.joins(query).db
}

func (s *DB) Includes(value interface{}) *DB {
	return s.clone().search.includes(value).db
}

func (s *DB) Scopes(funcs ...func(*DB) *DB) *DB {
	c := s
	for _, f := range funcs {
		c = f(c)
	}
	return c
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
	return s.clone().NewScope(out).Set("gorm:inline_condition", where).callCallbacks(s.parent.callback.queries).db
}

func (s *DB) Row() *sql.Row {
	return s.do(s.Value).row()
}

func (s *DB) Rows() (*sql.Rows, error) {
	return s.do(s.Value).rows()
}

func (s *DB) Scan(dest interface{}) *DB {
	return s.do(s.Value).query(dest).db
}

func (s *DB) Attrs(attrs ...interface{}) *DB {
	return s.clone().search.attrs(attrs...).db
}

func (s *DB) Assign(attrs ...interface{}) *DB {
	return s.clone().search.assign(attrs...).db
}

func (s *DB) FirstOrInit(out interface{}, where ...interface{}) *DB {
	c := s.clone()
	if c.First(out, where...).Error == RecordNotFound {
		return c.do(out).where(where).initialize().db
	} else if len(s.search.assignAttrs) > 0 {
		return c.do(out).updateAttrs(s.search.assignAttrs).db
	}
	return c
}

func (s *DB) FirstOrCreate(out interface{}, where ...interface{}) *DB {
	c := s.clone()
	if c.First(out, where...).Error == RecordNotFound {
		return c.do(out).where(where...).initialize().db.Save(out)
	} else if len(s.search.assignAttrs) > 0 {
		return c.do(out).updateAttrs(s.search.assignAttrs).update().db
	}
	return c
}

func (s *DB) Update(attrs ...interface{}) *DB {
	return s.Updates(toSearchableMap(attrs...), true)
}

func (s *DB) Updates(values interface{}, ignoreProtectedAttrs ...bool) *DB {
	return s.clone().NewScope(s.Value).
		Set("gorm:update_interface", values).
		Set("gorm:ignore_protected_attrs", len(ignoreProtectedAttrs) > 0).
		callCallbacks(s.parent.callback.updates).db
}

func (s *DB) UpdateColumn(attrs ...interface{}) *DB {
	return s.UpdateColumns(toSearchableMap(attrs...))
}

func (s *DB) UpdateColumns(values interface{}) *DB {
	return s.clone().NewScope(s.Value).
		Set("gorm:update_interface", values).
		Set("gorm:update_column", true).
		callCallbacks(s.parent.callback.updates).db
}

func (s *DB) Save(value interface{}) *DB {
	scope := s.clone().NewScope(value)
	if scope.PrimaryKeyZero() {
		return scope.callCallbacks(s.parent.callback.creates).db
	} else {
		return scope.callCallbacks(s.parent.callback.updates).db
	}
}

func (s *DB) Delete(value interface{}) *DB {
	return s.clone().NewScope(value).callCallbacks(s.parent.callback.deletes).db
}

func (s *DB) Raw(sql string, values ...interface{}) *DB {
	return s.clone().search.setraw(true).where(sql, values...).db
}

func (s *DB) Exec(sql string, values ...interface{}) *DB {
	return s.clone().do(nil).raw(sql, values...).exec().db
}

func (s *DB) Model(value interface{}) *DB {
	c := s.clone()
	c.Value = value
	return c
}

func (s *DB) Related(value interface{}, foreign_keys ...string) *DB {
	old_data := s.Value
	return s.do(value).related(old_data, foreign_keys...).db
}

func (s *DB) Pluck(column string, value interface{}) *DB {
	return s.do(s.Value).pluck(column, value).db
}

func (s *DB) Count(value interface{}) *DB {
	return s.do(s.Value).count(value).db
}

func (s *DB) Table(name string) *DB {
	return s.clone().search.table(name).db
}

func (s *DB) Debug() *DB {
	return s.clone().LogMode(true)
}

func (s *DB) Begin() *DB {
	c := s.clone()
	if db, ok := c.db.(sqlDb); ok {
		tx, err := db.Begin()
		c.db = interface{}(tx).(sqlCommon)
		c.err(err)
	} else {
		c.err(CantStartTransaction)
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

func (s *DB) NewRecord(value interface{}) bool {
	return s.clone().do(value).model.primaryKeyZero()
}

func (s *DB) RecordNotFound() bool {
	return s.Error == RecordNotFound
}

// Migrations
func (s *DB) CreateTable(value interface{}) *DB {
	return s.clone().do(value).createTable().db
}

func (s *DB) DropTable(value interface{}) *DB {
	return s.clone().do(value).dropTable().db
}

func (s *DB) AutoMigrate(value interface{}) *DB {
	return s.clone().do(value).autoMigrate().db
}

func (s *DB) ModifyColumn(column string, typ string) *DB {
	s.clone().do(s.Value).modifyColumn(column, typ)
	return s
}

func (s *DB) DropColumn(column string) *DB {
	s.do(s.Value).dropColumn(column)
	return s
}

func (s *DB) AddIndex(column string, index_name ...string) *DB {
	s.clone().do(s.Value).addIndex(column, index_name...)
	return s
}

func (s *DB) RemoveIndex(column string) *DB {
	s.clone().do(s.Value).removeIndex(column)
	return s
}
