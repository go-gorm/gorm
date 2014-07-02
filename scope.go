package gorm

import (
	"errors"
	"fmt"
	"go/ast"
	"strings"
	"time"

	"reflect"
	"regexp"
)

type Scope struct {
	Value      interface{}
	Search     *search
	Sql        string
	SqlVars    []interface{}
	db         *DB
	_values    map[string]interface{}
	skipLeft   bool
	primaryKey string
}

// NewScope create scope for callbacks, including DB's search information
func (db *DB) NewScope(value interface{}) *Scope {
	db.Value = value
	return &Scope{db: db, Search: db.search, Value: value, _values: map[string]interface{}{}}
}

// New create a new Scope without search information
func (scope *Scope) New(value interface{}) *Scope {
	return &Scope{db: scope.db.parent, Search: &search{}, Value: value}
}

// NewDB create a new DB without search information
func (scope *Scope) NewDB() *DB {
	return scope.db.new()
}

// DB get *sql.DB
func (scope *Scope) DB() sqlCommon {
	return scope.db.db
}

// SkipLeft skip remaining callbacks
func (scope *Scope) SkipLeft() {
	scope.skipLeft = true
}

// Quote used to quote database column name according to database dialect
func (scope *Scope) Quote(str string) string {
	return scope.Dialect().Quote(str)
}

// Dialect get dialect
func (scope *Scope) Dialect() Dialect {
	return scope.db.parent.dialect
}

// Err write error
func (scope *Scope) Err(err error) error {
	if err != nil {
		scope.db.err(err)
	}
	return err
}

// Log print log message
func (scope *Scope) Log(v ...interface{}) {
	scope.db.log(v...)
}

// HasError check if there are any error
func (scope *Scope) HasError() bool {
	return scope.db.Error != nil
}

// PrimaryKey get the primary key's column name
func (scope *Scope) PrimaryKey() string {
	if scope.primaryKey != "" {
		return scope.primaryKey
	}

	scope.primaryKey = scope.getPrimaryKey()
	return scope.primaryKey
}

// PrimaryKeyZero check the primary key is blank or not
func (scope *Scope) PrimaryKeyZero() bool {
	return isBlank(reflect.ValueOf(scope.PrimaryKeyValue()))
}

// PrimaryKeyValue get the primary key's value
func (scope *Scope) PrimaryKeyValue() interface{} {
	data := reflect.Indirect(reflect.ValueOf(scope.Value))

	if data.Kind() == reflect.Struct {
		if field := data.FieldByName(snakeToUpperCamel(scope.PrimaryKey())); field.IsValid() {
			return field.Interface()
		}
	}
	return 0
}

// HasColumn to check if has column
func (scope *Scope) HasColumn(name string) bool {
	_, result := scope.FieldByName(name)
	return result
}

// FieldByName to get column's value and existence
func (scope *Scope) FieldByName(name string) (interface{}, bool) {
	data := reflect.Indirect(reflect.ValueOf(scope.Value))

	if data.Kind() == reflect.Struct {
		if field := data.FieldByName(name); field.IsValid() {
			return field.Interface(), true
		}
	} else if data.Kind() == reflect.Slice {
		return nil, reflect.New(data.Type().Elem()).Elem().FieldByName(name).IsValid()
	}
	return nil, false
}

// SetColumn to set the column's value
func (scope *Scope) SetColumn(column string, value interface{}) {
	if scope.Value == nil {
		return
	}

	data := reflect.Indirect(reflect.ValueOf(scope.Value))
	setFieldValue(data.FieldByName(snakeToUpperCamel(column)), value)
}

// CallMethod invoke method with necessary argument
func (scope *Scope) CallMethod(name string) {
	if scope.Value == nil {
		return
	}

	call := func(value interface{}) {
		if fm := reflect.ValueOf(value).MethodByName(name); fm.IsValid() {
			fi := fm.Interface()
			if f, ok := fi.(func()); ok {
				f()
			} else if f, ok := fi.(func(s *Scope)); ok {
				f(scope)
			} else if f, ok := fi.(func(s *DB)); ok {
				f(scope.db.new())
			} else if f, ok := fi.(func() error); ok {
				scope.Err(f())
			} else if f, ok := fi.(func(s *Scope) error); ok {
				scope.Err(f(scope))
			} else if f, ok := fi.(func(s *DB) error); ok {
				scope.Err(f(scope.db.new()))
			} else {
				scope.Err(errors.New(fmt.Sprintf("unsupported function %v", name)))
			}
		}
	}

	if values := reflect.Indirect(reflect.ValueOf(scope.Value)); values.Kind() == reflect.Slice {
		for i := 0; i < values.Len(); i++ {
			call(values.Index(i).Addr().Interface())
		}
	} else {
		call(scope.Value)
	}
}

// AddToVars add value as sql's vars, gorm will escape them
func (scope *Scope) AddToVars(value interface{}) string {
	scope.SqlVars = append(scope.SqlVars, value)
	return scope.Dialect().BinVar(len(scope.SqlVars))
}

// TableName get table name
var pluralMapKeys = []*regexp.Regexp{regexp.MustCompile("ch$"), regexp.MustCompile("ss$"), regexp.MustCompile("sh$"), regexp.MustCompile("day$"), regexp.MustCompile("y$"), regexp.MustCompile("x$"), regexp.MustCompile("([^s])s?$")}
var pluralMapValues = []string{"ches", "sses", "shes", "days", "ies", "xes", "${1}s"}

func (scope *Scope) TableName() string {
	if scope.Search != nil && len(scope.Search.TableName) > 0 {
		return scope.Search.TableName
	} else {
		if scope.Value == nil {
			scope.Err(errors.New("can't get table name"))
			return ""
		}
		data := reflect.Indirect(reflect.ValueOf(scope.Value))

		if data.Kind() == reflect.Slice {
			data = reflect.New(data.Type().Elem()).Elem()
		}

		if fm := data.MethodByName("TableName"); fm.IsValid() {
			if v := fm.Call([]reflect.Value{}); len(v) > 0 {
				if result, ok := v[0].Interface().(string); ok {
					return result
				}
			}
		}

		str := toSnake(data.Type().Name())

		if !scope.db.parent.singularTable {
			for index, reg := range pluralMapKeys {
				if reg.MatchString(str) {
					return reg.ReplaceAllString(str, pluralMapValues[index])
				}
			}
		}

		return str
	}
}

func (scope *Scope) QuotedTableName() string {
	if scope.Search != nil && len(scope.Search.TableName) > 0 {
		return scope.Search.TableName
	} else {
		keys := strings.Split(scope.TableName(), ".")
		for i, v := range keys {
			keys[i] = scope.Quote(v)
		}
		return strings.Join(keys, ".")
	}
}

// CombinedConditionSql get combined condition sql
func (scope *Scope) CombinedConditionSql() string {
	return scope.joinsSql() + scope.whereSql() + scope.groupSql() +
		scope.havingSql() + scope.orderSql() + scope.limitSql() + scope.offsetSql()
}

// Fields get value's fields
func (scope *Scope) Fields() []*Field {
	indirectValue := reflect.Indirect(reflect.ValueOf(scope.Value))
	fields := []*Field{}

	if !indirectValue.IsValid() {
		return fields
	}

	scopeTyp := indirectValue.Type()
	for i := 0; i < scopeTyp.NumField(); i++ {
		fieldStruct := scopeTyp.Field(i)
		if !ast.IsExported(fieldStruct.Name) {
			continue
		}

		var field Field
		field.Name = fieldStruct.Name
		field.DBName = toSnake(fieldStruct.Name)

		value := indirectValue.FieldByName(fieldStruct.Name)
		field.Value = value.Interface()
		field.IsBlank = isBlank(value)

		// Search for primary key tag identifier
		field.isPrimaryKey = scope.PrimaryKey() == field.DBName || fieldStruct.Tag.Get("primaryKey") != ""

		if field.isPrimaryKey {
			scope.primaryKey = field.DBName
		}

		if scope.db != nil {
			field.Tag = fieldStruct.Tag
			field.SqlTag = scope.sqlTagForField(&field)

			// parse association
			elem := reflect.Indirect(value)
			typ := elem.Type()

			switch elem.Kind() {
			case reflect.Slice:
				typ = typ.Elem()

				if typ.Kind() == reflect.Struct {
					foreignKey := scopeTyp.Name() + "Id"
					if reflect.New(typ).Elem().FieldByName(foreignKey).IsValid() {
						field.ForeignKey = foreignKey
					}
					field.AfterAssociation = true
				}
			case reflect.Struct:
				if !field.IsTime() && !field.IsScanner() {
					if scope.HasColumn(field.Name + "Id") {
						field.ForeignKey = field.Name + "Id"
						field.BeforeAssociation = true
					} else {
						foreignKey := scopeTyp.Name() + "Id"
						if reflect.New(typ).Elem().FieldByName(foreignKey).IsValid() {
							field.ForeignKey = foreignKey
						}
						field.AfterAssociation = true
					}
				}
			}
		}
		fields = append(fields, &field)
	}

	return fields
}

// Raw set sql
func (scope *Scope) Raw(sql string) *Scope {
	scope.Sql = strings.Replace(sql, "$$", "?", -1)
	return scope
}

// Exec invoke sql
func (scope *Scope) Exec() *Scope {
	defer scope.Trace(time.Now())

	if !scope.HasError() {
		result, err := scope.DB().Exec(scope.Sql, scope.SqlVars...)
		if scope.Err(err) == nil {
			if count, err := result.RowsAffected(); err == nil {
				scope.db.RowsAffected = count
			}
		}
	}
	return scope
}

// Set set value by name
func (scope *Scope) Set(name string, value interface{}) *Scope {
	scope._values[name] = value
	return scope
}

// Get get value by name
func (scope *Scope) Get(name string) (value interface{}, ok bool) {
	value, ok = scope._values[name]
	return
}

// Trace print sql log
func (scope *Scope) Trace(t time.Time) {
	if len(scope.Sql) > 0 {
		scope.db.slog(scope.Sql, t, scope.SqlVars...)
	}
}

// Begin start a transaction
func (scope *Scope) Begin() *Scope {
	if db, ok := scope.DB().(sqlDb); ok {
		if tx, err := db.Begin(); err == nil {
			scope.db.db = interface{}(tx).(sqlCommon)
			scope.Set("gorm:started_transaction", true)
		}
	}
	return scope
}

// CommitOrRollback commit current transaction if there is no error, otherwise rollback it
func (scope *Scope) CommitOrRollback() *Scope {
	if _, ok := scope.Get("gorm:started_transaction"); ok {
		if db, ok := scope.db.db.(sqlTx); ok {
			if scope.HasError() {
				db.Rollback()
			} else {
				db.Commit()
			}
			scope.db.db = scope.db.parent.db
		}
	}
	return scope
}
