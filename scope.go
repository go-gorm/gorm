package gorm

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm/dialect"
	"go/ast"
	"strings"
	"time"

	"reflect"
	"regexp"
)

type Scope struct {
	Value   interface{}
	Search  *search
	Sql     string
	SqlVars []interface{}
	db      *DB
	_values map[string]interface{}
}

func (db *DB) NewScope(value interface{}) *Scope {
	db.Value = value
	return &Scope{db: db, Search: db.search, Value: value, _values: map[string]interface{}{}}
}

func (scope *Scope) callCallbacks(funcs []*func(s *Scope)) *Scope {
	for _, f := range funcs {
		(*f)(scope)
	}
	return scope
}

func (scope *Scope) New(value interface{}) *Scope {
	return &Scope{db: scope.db.parent, Search: &search{}, Value: value}
}

func (scope *Scope) NewDB() *DB {
	return scope.db.new()
}

func (scope *Scope) DB() sqlCommon {
	return scope.db.db
}

func (scope *Scope) Dialect() dialect.Dialect {
	return scope.db.parent.dialect
}

func (scope *Scope) Err(err error) error {
	if err != nil {
		scope.db.err(err)
	}
	return err
}

func (scope *Scope) HasError() bool {
	return scope.db.hasError()
}

func (scope *Scope) PrimaryKey() string {
	return "id"
}

func (scope *Scope) PrimaryKeyZero() bool {
	return isBlank(reflect.ValueOf(scope.PrimaryKeyValue()))
}

func (scope *Scope) PrimaryKeyValue() interface{} {
	data := reflect.Indirect(reflect.ValueOf(scope.Value))

	if data.Kind() == reflect.Struct {
		if field := data.FieldByName(snakeToUpperCamel(scope.PrimaryKey())); field.IsValid() {
			return field.Interface()
		}
	}
	return 0
}

func (scope *Scope) HasColumn(name string) bool {
	data := reflect.Indirect(reflect.ValueOf(scope.Value))

	if data.Kind() == reflect.Struct {
		return data.FieldByName(name).IsValid()
	} else if data.Kind() == reflect.Slice {
		return reflect.New(data.Type().Elem()).Elem().FieldByName(name).IsValid()
	}
	return false
}

func (scope *Scope) SetColumn(column string, value interface{}) {
	data := reflect.Indirect(reflect.ValueOf(scope.Value))
	setFieldValue(data.FieldByName(snakeToUpperCamel(column)), value)
}

func (scope *Scope) CallMethod(name string) {
	if fm := reflect.ValueOf(scope.Value).MethodByName(name); fm.IsValid() {
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

func (scope *Scope) AddToVars(value interface{}) string {
	scope.SqlVars = append(scope.SqlVars, value)
	return scope.Dialect().BinVar(len(scope.SqlVars))
}

func (scope *Scope) TableName() string {
	if len(scope.Search.tableName) > 0 {
		return scope.Search.tableName
	} else {
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
			pluralMap := map[string]string{"ch": "ches", "ss": "sses", "sh": "shes", "day": "days", "y": "ies", "x": "xes", "s?": "s"}
			for key, value := range pluralMap {
				reg := regexp.MustCompile(key + "$")
				if reg.MatchString(str) {
					return reg.ReplaceAllString(str, value)
				}
			}
		}

		return str
	}
}

func (s *Scope) CombinedConditionSql() string {
	return s.joinsSql() + s.whereSql() + s.groupSql() + s.havingSql() + s.orderSql() + s.limitSql() + s.offsetSql()
}

func (scope *Scope) SqlTagForField(field *Field) (tag string) {
	value := field.Value
	reflect_value := reflect.ValueOf(value)

	if field.IsScanner() {
		value = reflect_value.Field(0).Interface()
	}

	switch reflect_value.Kind() {
	case reflect.Slice:
		if _, ok := value.([]byte); !ok {
			return
		}
	case reflect.Struct:
		if !field.IsTime() && !field.IsScanner() {
			return
		}
	}

	if tag = field.Tag; len(tag) == 0 && tag != "-" {
		if field.isPrimaryKey {
			tag = scope.Dialect().PrimaryKeyTag(value, field.Size)
		} else {
			tag = scope.Dialect().SqlTag(value, field.Size)
		}

		if len(field.AddationalTag) > 0 {
			tag = tag + " " + field.AddationalTag
		}
	}
	return
}

func (scope *Scope) Fields() []*Field {
	indirect_value := reflect.Indirect(reflect.ValueOf(scope.Value))
	fields := []*Field{}

	if !indirect_value.IsValid() {
		return fields
	}

	scope_typ := indirect_value.Type()
	for i := 0; i < scope_typ.NumField(); i++ {
		field_struct := scope_typ.Field(i)
		if field_struct.Anonymous || !ast.IsExported(field_struct.Name) {
			continue
		}

		var field Field
		field.Name = field_struct.Name
		field.DBName = toSnake(field_struct.Name)

		value := indirect_value.FieldByName(field_struct.Name)
		field.Value = value.Interface()
		field.IsBlank = isBlank(value)

		tag, addational_tag, size := parseSqlTag(field_struct.Tag.Get(scope.db.parent.tagIdentifier))
		field.Tag = tag
		field.AddationalTag = addational_tag
		field.Size = size
		field.SqlTag = scope.SqlTagForField(&field)

		if tag == "-" {
			field.IsIgnored = true
		}

		// parse association
		elem := reflect.Indirect(value)
		typ := elem.Type()

		switch elem.Kind() {
		case reflect.Slice:
			typ = typ.Elem()

			if _, ok := field.Value.([]byte); !ok {
				foreignKey := scope_typ.Name() + "Id"
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
					foreignKey := scope_typ.Name() + "Id"
					if reflect.New(typ).Elem().FieldByName(foreignKey).IsValid() {
						field.ForeignKey = foreignKey
					}
					field.AfterAssociation = true
				}
			}
		}
		fields = append(fields, &field)
	}

	return fields
}

func (scope *Scope) Raw(sql string) {
	scope.Sql = strings.Replace(sql, "$$", "?", -1)
}

func (scope *Scope) Exec() {
	if !scope.HasError() {
		_, err := scope.DB().Exec(scope.Sql, scope.SqlVars...)
		scope.Err(err)
	}
}

func (scope *Scope) Get(name string) (value interface{}, ok bool) {
	value, ok = scope._values[name]
	return
}

func (scope *Scope) Set(name string, value interface{}) {
	scope._values[name] = value
}

func (scope *Scope) Trace(t time.Time) {
	if len(scope.Sql) > 0 {
		scope.db.slog(scope.Sql, t, scope.SqlVars...)
	}
}

func (scope *Scope) Begin() *Scope {
	if db, ok := scope.DB().(sqlDb); ok {
		if tx, err := db.Begin(); err == nil {
			scope.db.db = interface{}(tx).(sqlCommon)
			scope.Set("gorm:started_transaction", true)
		}
	}
	return scope
}

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
