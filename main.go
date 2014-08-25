package gorm

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
)

type DB struct {
	Value         interface{}
	Error         error
	RowsAffected  int64
	callback      *callback
	db            sqlCommon
	parent        *DB
	search        *search
	logMode       int
	logger        logger
	dialect       Dialect
	tagIdentifier string
	singularTable bool
	source        string
	values        map[string]interface{}
}

func Open(dialect string, drivesources ...string) (DB, error) {
	var db DB
	var err error
	var driver = dialect
	var source string

	if len(drivesources) == 0 {
		err = errors.New("invalid database source")
	} else {
		if len(drivesources) == 1 {
			source = drivesources[0]
		} else if len(drivesources) >= 2 {
			driver = drivesources[0]
			source = drivesources[1]
		}

		db = DB{dialect: NewDialect(dialect), tagIdentifier: "sql", logger: defaultLogger, callback: DefaultCallback, source: source, values: map[string]interface{}{}}
		db.db, err = sql.Open(driver, source)
		db.parent = &db
	}
	return db, err
}

func (s *DB) Close() error {
	return s.parent.db.(*sql.DB).Close()
}

func (s *DB) DB() *sql.DB {
	return s.db.(*sql.DB)
}

// Return the underlying sql.DB or sql.Tx instance.
// Use of this method is discouraged. It's mainly intended to allow
// coexistence with legacy non-GORM code.
func (s *DB) CommonDB() sqlCommon {
	return s.db
}

func (s *DB) Callback() *callback {
	s.parent.callback = s.parent.callback.clone()
	return s.parent.callback
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

func (s *DB) Attrs(attrs ...interface{}) *DB {
	return s.clone().search.attrs(attrs...).db
}

func (s *DB) Assign(attrs ...interface{}) *DB {
	return s.clone().search.assign(attrs...).db
}

func (s *DB) First(out interface{}, where ...interface{}) *DB {
	return s.clone().Limit(1).NewScope(out).InstanceSet("gorm:order_by_primary_key", "ASC").
		inlineCondition(where...).callCallbacks(s.parent.callback.queries).db
}

func (s *DB) Last(out interface{}, where ...interface{}) *DB {
	return s.clone().Limit(1).NewScope(out).InstanceSet("gorm:order_by_primary_key", "DESC").
		inlineCondition(where...).callCallbacks(s.parent.callback.queries).db
}

func (s *DB) Find(out interface{}, where ...interface{}) *DB {
	return s.clone().NewScope(out).inlineCondition(where...).callCallbacks(s.parent.callback.queries).db
}

func (s *DB) Row() *sql.Row {
	return s.NewScope(s.Value).row()
}

func (s *DB) Rows() (*sql.Rows, error) {
	return s.NewScope(s.Value).rows()
}

func (s *DB) Scan(dest interface{}) *DB {
	scope := s.clone().NewScope(s.Value).InstanceSet("gorm:query_destination", dest)
	Query(scope)
	return scope.db
}

func (s *DB) FirstOrInit(out interface{}, where ...interface{}) *DB {
	c := s.clone()
	r := c.First(out, where...)
	if r.Error != nil {
		if !r.RecordNotFound() {
			return r
		}
		c.NewScope(out).inlineCondition(where...).initialize()
	} else {
		c.NewScope(out).updatedAttrsWithValues(convertInterfaceToMap(s.search.AssignAttrs), false)
	}
	return c
}

func (s *DB) FirstOrCreate(out interface{}, where ...interface{}) *DB {
	c := s.clone()
	r := c.First(out, where...)
	if r.Error != nil {
		if !r.RecordNotFound() {
			return r
		}
		c.NewScope(out).inlineCondition(where...).initialize().callCallbacks(s.parent.callback.creates)
	} else if len(c.search.AssignAttrs) > 0 {
		c.NewScope(out).InstanceSet("gorm:update_interface", s.search.AssignAttrs).callCallbacks(s.parent.callback.updates)
	}
	return c
}

func (s *DB) Update(attrs ...interface{}) *DB {
	return s.Updates(toSearchableMap(attrs...), true)
}

func (s *DB) Updates(values interface{}, ignoreProtectedAttrs ...bool) *DB {
	return s.clone().NewScope(s.Value).
		Set("gorm:ignore_protected_attrs", len(ignoreProtectedAttrs) > 0).
		InstanceSet("gorm:update_interface", values).
		callCallbacks(s.parent.callback.updates).db
}

func (s *DB) UpdateColumn(attrs ...interface{}) *DB {
	return s.UpdateColumns(toSearchableMap(attrs...))
}

func (s *DB) UpdateColumns(values interface{}) *DB {
	return s.clone().NewScope(s.Value).
		Set("gorm:update_column", true).
		InstanceSet("gorm:update_interface", values).
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

func (s *DB) Create(value interface{}) *DB {
	scope := s.clone().NewScope(value)
	return scope.callCallbacks(s.parent.callback.creates).db
}

func (s *DB) Delete(value interface{}, where ...interface{}) *DB {
	return s.clone().NewScope(value).inlineCondition(where...).callCallbacks(s.parent.callback.deletes).db
}

func (s *DB) Raw(sql string, values ...interface{}) *DB {
	return s.clone().search.raw(true).where(sql, values...).db
}

func (s *DB) Exec(sql string, values ...interface{}) *DB {
	scope := s.clone().NewScope(nil)
	scope.Raw(scope.buildWhereCondition(map[string]interface{}{"query": sql, "args": values}))
	return scope.Exec().db
}

func (s *DB) Model(value interface{}) *DB {
	c := s.clone()
	c.Value = value
	return c
}

func (s *DB) Related(value interface{}, foreignKeys ...string) *DB {
	return s.clone().NewScope(s.Value).related(value, foreignKeys...).db
}

func (s *DB) Pluck(column string, value interface{}) *DB {
	return s.NewScope(s.Value).pluck(column, value).db
}

func (s *DB) Count(value interface{}) *DB {
	return s.NewScope(s.Value).count(value).db
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
	return s.clone().NewScope(value).PrimaryKeyZero()
}

func (s *DB) RecordNotFound() bool {
	return s.Error == RecordNotFound
}

// Migrations
func (s *DB) CreateTable(value interface{}) *DB {
	return s.clone().NewScope(value).createTable().db
}

func (s *DB) DropTable(value interface{}) *DB {
	return s.clone().NewScope(value).dropTable().db
}

func (s *DB) DropTableIfExists(value interface{}) *DB {
	return s.clone().NewScope(value).dropTableIfExists().db
}

func (s *DB) AutoMigrate(values ...interface{}) *DB {
	db := s.clone()
	for _, value := range values {
		db = db.NewScope(value).autoMigrate().db
	}
	return db
}

func (s *DB) ModifyColumn(column string, typ string) *DB {
	s.clone().NewScope(s.Value).modifyColumn(column, typ)
	return s
}

func (s *DB) DropColumn(column string) *DB {
	s.clone().NewScope(s.Value).dropColumn(column)
	return s
}

func (s *DB) AddIndex(indexName string, column ...string) *DB {
	s.clone().NewScope(s.Value).addIndex(false, indexName, column...)
	return s
}

func (s *DB) AddUniqueIndex(indexName string, column ...string) *DB {
	s.clone().NewScope(s.Value).addIndex(true, indexName, column...)
	return s
}

func (s *DB) RemoveIndex(indexName string) *DB {
	s.clone().NewScope(s.Value).removeIndex(indexName)
	return s
}

func (s *DB) Association(column string) *Association {
	scope := s.clone().NewScope(s.Value)

	primaryKey := scope.PrimaryKeyValue()
	if reflect.DeepEqual(reflect.ValueOf(primaryKey), reflect.Zero(reflect.ValueOf(primaryKey).Type())) {
		scope.Err(errors.New("primary key can't be nil"))
	}

	var field *Field
	scopeType := scope.IndirectValue().Type()
	if f, ok := scopeType.FieldByName(SnakeToUpperCamel(column)); ok {
		field = scope.fieldFromStruct(f)
		if field.Relationship == nil || field.Relationship.ForeignKey == "" {
			scope.Err(fmt.Errorf("invalid association %v for %v", column, scopeType))
		}
	} else {
		scope.Err(fmt.Errorf("%v doesn't have column %v", scopeType, column))
	}

	return &Association{Scope: scope, Column: column, Error: s.Error, PrimaryKey: primaryKey, Field: field}
}

// Set set value by name
func (s *DB) Set(name string, value interface{}) *DB {
	return s.clone().InstantSet(name, value)
}

func (s *DB) InstantSet(name string, value interface{}) *DB {
	s.values[name] = value
	return s
}

// Get get value by name
func (s *DB) Get(name string) (value interface{}, ok bool) {
	value, ok = s.values[name]
	return
}
