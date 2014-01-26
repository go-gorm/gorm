package gorm

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm/dialect"
	"strings"

	"reflect"
	"regexp"
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

func (scope *Scope) CombinedConditionSql() string {
	return ""
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
