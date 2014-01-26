package gorm

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm/dialect"

	"reflect"
)

type Scope struct {
	Value   interface{}
	Search  *search
	Sql     string
	SqlVars []interface{}
	db      *DB
}

func (db *DB) newScope(value interface{}) *Scope {
	return &Scope{db: db, Search: db.search, Value: value}
}

func (scope *Scope) callCallbacks(funcs []*func(s *Scope)) *Scope {
	for _, f := range funcs {
		(*f)(scope)
	}
	return scope
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
	return "Id"
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
	} else {
		scope.Err(errors.New(fmt.Sprintf("no valid function %v found", name)))
	}
}

func (scope *Scope) CombinedConditionSql() string {
	return ""
}

func (scope *Scope) AddToVars(value interface{}) string {
	return ""
}

func (scope *Scope) TableName() string {
	return ""
}

func (scope *Scope) Raw(sql string, values ...interface{}) {
	fmt.Println(sql, values)
}

func (scope *Scope) Exec() {
}
