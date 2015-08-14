package gorm

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// NowFunc returns current time, this function is exported in order to be able
// to give the flexibility to the developer to customize it according to their
// needs
//
//   e.g: return time.Now().UTC()
//
var NowFunc = func() time.Time {
	return time.Now()
}

type DB struct {
	Value             interface{}
	Error             error
	RowsAffected      int64
	callback          *callback
	db                sqlCommon
	parent            *DB
	search            *search
	logMode           int
	logger            logger
	dialect           Dialect
	singularTable     bool
	source            string
	values            map[string]interface{}
	joinTableHandlers map[string]JoinTableHandler
}

func Open(dialect string, args ...interface{}) (DB, error) {
	var db DB
	var err error

	if len(args) == 0 {
		err = errors.New("invalid database source")
	} else {
		var source string
		var dbSql sqlCommon

		switch value := args[0].(type) {
		case string:
			var driver = dialect
			if len(args) == 1 {
				source = value
			} else if len(args) >= 2 {
				driver = value
				source = args[1].(string)
			}
			if driver == "foundation" {
				driver = "postgres" // FoundationDB speaks a postgres-compatible protocol.
			}
			dbSql, err = sql.Open(driver, source)
		case sqlCommon:
			source = reflect.Indirect(reflect.ValueOf(value)).FieldByName("dsn").String()
			dbSql = value
		}

		db = DB{
			dialect:  NewDialect(dialect),
			logger:   defaultLogger,
			callback: DefaultCallback,
			source:   source,
			values:   map[string]interface{}{},
			db:       dbSql,
		}
		db.parent = &db

		if err == nil {
			err = db.DB().Ping() // Send a ping to make sure the database connection is alive.
		}
	}

	return db, err
}

func (s *DB) Close() error {
	return s.parent.db.(*sql.DB).Close()
}

func (s *DB) DB() *sql.DB {
	return s.db.(*sql.DB)
}

func (s *DB) New() *DB {
	clone := s.clone()
	clone.search = nil
	clone.Value = nil
	return clone
}

// NewScope create scope for callbacks, including DB's search information
func (db *DB) NewScope(value interface{}) *Scope {
	dbClone := db.clone()
	dbClone.Value = value
	return &Scope{db: dbClone, Search: dbClone.search.clone(), Value: value}
}

// CommonDB Return the underlying sql.DB or sql.Tx instance.
// Use of this method is discouraged. It's mainly intended to allow
// coexistence with legacy non-GORM code.
func (s *DB) CommonDB() sqlCommon {
	return s.db
}

func (s *DB) Callback() *callback {
	s.parent.callback = s.parent.callback.clone()
	return s.parent.callback
}

func (s *DB) SetLogger(l logger) {
	s.logger = l
}

func (s *DB) LogMode(enable bool) *DB {
	if enable {
		s.logMode = 2
	} else {
		s.logMode = 1
	}
	return s
}

func (s *DB) SingularTable(enable bool) {
	modelStructs = map[reflect.Type]*ModelStruct{}
	s.parent.singularTable = enable
}

func (s *DB) Where(query interface{}, args ...interface{}) *DB {
	return s.clone().search.Where(query, args...).db
}

func (s *DB) Or(query interface{}, args ...interface{}) *DB {
	return s.clone().search.Or(query, args...).db
}

func (s *DB) Not(query interface{}, args ...interface{}) *DB {
	return s.clone().search.Not(query, args...).db
}

func (s *DB) Limit(value interface{}) *DB {
	return s.clone().search.Limit(value).db
}

func (s *DB) Offset(value interface{}) *DB {
	return s.clone().search.Offset(value).db
}

func (s *DB) Order(value string, reorder ...bool) *DB {
	return s.clone().search.Order(value, reorder...).db
}

func (s *DB) Select(query interface{}, args ...interface{}) *DB {
	return s.clone().search.Select(query, args...).db
}

func (s *DB) Omit(columns ...string) *DB {
	return s.clone().search.Omit(columns...).db
}

func (s *DB) Group(query string) *DB {
	return s.clone().search.Group(query).db
}

func (s *DB) Having(query string, values ...interface{}) *DB {
	return s.clone().search.Having(query, values...).db
}

func (s *DB) Joins(query string) *DB {
	return s.clone().search.Joins(query).db
}

func (s *DB) Scopes(funcs ...func(*DB) *DB) *DB {
	for _, f := range funcs {
		s = f(s)
	}
	return s
}

func (s *DB) Unscoped() *DB {
	return s.clone().search.unscoped().db
}

func (s *DB) Attrs(attrs ...interface{}) *DB {
	return s.clone().search.Attrs(attrs...).db
}

func (s *DB) Assign(attrs ...interface{}) *DB {
	return s.clone().search.Assign(attrs...).db
}

func (s *DB) First(out interface{}, where ...interface{}) *DB {
	newScope := s.clone().NewScope(out)
	newScope.Search.Limit(1)
	return newScope.Set("gorm:order_by_primary_key", "ASC").
		inlineCondition(where...).callCallbacks(s.parent.callback.queries).db
}

func (s *DB) Last(out interface{}, where ...interface{}) *DB {
	newScope := s.clone().NewScope(out)
	newScope.Search.Limit(1)
	return newScope.Set("gorm:order_by_primary_key", "DESC").
		inlineCondition(where...).callCallbacks(s.parent.callback.queries).db
}

func (s *DB) Find(out interface{}, where ...interface{}) *DB {
	return s.clone().NewScope(out).inlineCondition(where...).callCallbacks(s.parent.callback.queries).db
}

func (s *DB) Scan(dest interface{}) *DB {
	return s.clone().NewScope(s.Value).Set("gorm:query_destination", dest).callCallbacks(s.parent.callback.queries).db
}

func (s *DB) Row() *sql.Row {
	return s.NewScope(s.Value).row()
}

func (s *DB) Rows() (*sql.Rows, error) {
	return s.NewScope(s.Value).rows()
}

func (s *DB) Pluck(column string, value interface{}) *DB {
	return s.NewScope(s.Value).pluck(column, value).db
}

func (s *DB) Count(value interface{}) *DB {
	return s.NewScope(s.Value).count(value).db
}

func (s *DB) Related(value interface{}, foreignKeys ...string) *DB {
	return s.clone().NewScope(s.Value).related(value, foreignKeys...).db
}

func (s *DB) FirstOrInit(out interface{}, where ...interface{}) *DB {
	c := s.clone()
	if result := c.First(out, where...); result.Error != nil {
		if !result.RecordNotFound() {
			return result
		}
		c.NewScope(out).inlineCondition(where...).initialize()
	} else {
		c.NewScope(out).updatedAttrsWithValues(convertInterfaceToMap(s.search.assignAttrs), false)
	}
	return c
}

func (s *DB) FirstOrCreate(out interface{}, where ...interface{}) *DB {
	c := s.clone()
	if result := c.First(out, where...); result.Error != nil {
		if !result.RecordNotFound() {
			return result
		}
		c.AddError(c.NewScope(out).inlineCondition(where...).initialize().callCallbacks(s.parent.callback.creates).db.Error)
	} else if len(c.search.assignAttrs) > 0 {
		c.AddError(c.NewScope(out).InstanceSet("gorm:update_interface", s.search.assignAttrs).callCallbacks(s.parent.callback.updates).db.Error)
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
		Set("gorm:save_associations", false).
		InstanceSet("gorm:update_interface", values).
		callCallbacks(s.parent.callback.updates).db
}

func (s *DB) Save(value interface{}) *DB {
	scope := s.clone().NewScope(value)
	if scope.PrimaryKeyZero() {
		return scope.callCallbacks(s.parent.callback.creates).db
	}
	return scope.callCallbacks(s.parent.callback.updates).db
}

func (s *DB) Create(value interface{}) *DB {
	scope := s.clone().NewScope(value)
	return scope.callCallbacks(s.parent.callback.creates).db
}

func (s *DB) Delete(value interface{}, where ...interface{}) *DB {
	return s.clone().NewScope(value).inlineCondition(where...).callCallbacks(s.parent.callback.deletes).db
}

func (s *DB) Raw(sql string, values ...interface{}) *DB {
	return s.clone().search.Raw(true).Where(sql, values...).db
}

func (s *DB) Exec(sql string, values ...interface{}) *DB {
	scope := s.clone().NewScope(nil)
	generatedSql := scope.buildWhereCondition(map[string]interface{}{"query": sql, "args": values})
	generatedSql = strings.TrimSuffix(strings.TrimPrefix(generatedSql, "("), ")")
	scope.Raw(generatedSql)
	return scope.Exec().db
}

func (s *DB) Model(value interface{}) *DB {
	c := s.clone()
	c.Value = value
	return c
}

func (s *DB) Table(name string) *DB {
	clone := s.clone()
	clone.search.Table(name)
	clone.Value = nil
	return clone
}

func (s *DB) Debug() *DB {
	return s.clone().LogMode(true)
}

func (s *DB) Begin() *DB {
	c := s.clone()
	if db, ok := c.db.(sqlDb); ok {
		tx, err := db.Begin()
		c.db = interface{}(tx).(sqlCommon)
		c.AddError(err)
	} else {
		c.AddError(CantStartTransaction)
	}
	return c
}

func (s *DB) Commit() *DB {
	if db, ok := s.db.(sqlTx); ok {
		s.AddError(db.Commit())
	} else {
		s.AddError(NoValidTransaction)
	}
	return s
}

func (s *DB) Rollback() *DB {
	if db, ok := s.db.(sqlTx); ok {
		s.AddError(db.Rollback())
	} else {
		s.AddError(NoValidTransaction)
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

func (s *DB) HasTable(value interface{}) bool {
	scope := s.clone().NewScope(value)
	tableName := scope.TableName()
	has := scope.Dialect().HasTable(scope, tableName)
	s.AddError(scope.db.Error)
	return has
}

func (s *DB) AutoMigrate(values ...interface{}) *DB {
	db := s.clone()
	for _, value := range values {
		db = db.NewScope(value).NeedPtr().autoMigrate().db
	}
	return db
}

func (s *DB) ModifyColumn(column string, typ string) *DB {
	scope := s.clone().NewScope(s.Value)
	scope.modifyColumn(column, typ)
	return scope.db
}

func (s *DB) DropColumn(column string) *DB {
	scope := s.clone().NewScope(s.Value)
	scope.dropColumn(column)
	return scope.db
}

func (s *DB) AddIndex(indexName string, column ...string) *DB {
	scope := s.clone().NewScope(s.Value)
	scope.addIndex(false, indexName, column...)
	return scope.db
}

func (s *DB) AddUniqueIndex(indexName string, column ...string) *DB {
	scope := s.clone().NewScope(s.Value)
	scope.addIndex(true, indexName, column...)
	return scope.db
}

func (s *DB) RemoveIndex(indexName string) *DB {
	scope := s.clone().NewScope(s.Value)
	scope.removeIndex(indexName)
	return scope.db
}

func (s *DB) CurrentDatabase() string {
	var (
		scope = s.clone().NewScope(s.Value)
		name  = s.dialect.CurrentDatabase(scope)
	)
	return name
}

/*
Add foreign key to the given scope

Example:
	db.Model(&User{}).AddForeignKey("city_id", "cities(id)", "RESTRICT", "RESTRICT")
*/
func (s *DB) AddForeignKey(field string, dest string, onDelete string, onUpdate string) *DB {
	scope := s.clone().NewScope(s.Value)
	scope.addForeignKey(field, dest, onDelete, onUpdate)
	return scope.db
}

func (s *DB) Association(column string) *Association {
	var err error
	scope := s.clone().NewScope(s.Value)

	if primaryField := scope.PrimaryField(); primaryField.IsBlank {
		err = errors.New("primary key can't be nil")
	} else {
		if field, ok := scope.FieldByName(column); ok {
			if field.Relationship == nil || len(field.Relationship.ForeignFieldNames) == 0 {
				err = fmt.Errorf("invalid association %v for %v", column, scope.IndirectValue().Type())
			} else {
				return &Association{Scope: scope, Column: column, Field: field}
			}
		} else {
			err = fmt.Errorf("%v doesn't have column %v", scope.IndirectValue().Type(), column)
		}
	}

	return &Association{Error: err}
}

func (s *DB) Preload(column string, conditions ...interface{}) *DB {
	return s.clone().search.Preload(column, conditions...).db
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

func (s *DB) SetJoinTableHandler(source interface{}, column string, handler JoinTableHandlerInterface) {
	scope := s.NewScope(source)
	for _, field := range scope.GetModelStruct().StructFields {
		if field.Name == column || field.DBName == column {
			if many2many := parseTagSetting(field.Tag.Get("gorm"))["MANY2MANY"]; many2many != "" {
				source := (&Scope{Value: source}).GetModelStruct().ModelType
				destination := (&Scope{Value: reflect.New(field.Struct.Type).Interface()}).GetModelStruct().ModelType
				handler.Setup(field.Relationship, many2many, source, destination)
				field.Relationship.JoinTableHandler = handler
				if table := handler.Table(s); scope.Dialect().HasTable(scope, table) {
					s.Table(table).AutoMigrate(handler)
				}
			}
		}
	}
}

func (s *DB) AddError(err error) error {
	if err != nil {
		if err != RecordNotFound {
			if s.logMode == 0 {
				go s.print(fileWithLineNum(), err)
			} else {
				s.log(err)
			}

			if e, ok := err.(errorsInterface); ok {
				err = Errors{errors: append(s.GetErrors(), e.GetErrors()...)}
			} else {
				err = Errors{errors: append(s.GetErrors(), err)}
			}
		}

		s.Error = err
	}
	return err
}

func (s *DB) GetErrors() (errors []error) {
	if errs, ok := s.Error.(errorsInterface); ok {
		return errs.GetErrors()
	} else if s.Error != nil {
		return []error{s.Error}
	}
	return
}
