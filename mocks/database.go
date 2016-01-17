package mocks

import "github.com/jinzhu/gorm"
import "github.com/stretchr/testify/mock"

import (
	"database/sql"
)

type Database struct {
	mock.Mock
}

func (_m *Database) Close() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Database) DB() *sql.DB {
	ret := _m.Called()

	var r0 *sql.DB
	if rf, ok := ret.Get(0).(func() *sql.DB); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sql.DB)
		}
	}

	return r0
}
func (_m *Database) New() gorm.Database {
	ret := _m.Called()

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func() gorm.Database); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) NewScope(value interface{}) *gorm.Scope {
	ret := _m.Called(value)

	var r0 *gorm.Scope
	if rf, ok := ret.Get(0).(func(interface{}) *gorm.Scope); ok {
		r0 = rf(value)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*gorm.Scope)
		}
	}

	return r0
}
func (_m *Database) CommonDB() gorm.SqlCommon {
	ret := _m.Called()

	var r0 gorm.SqlCommon
	if rf, ok := ret.Get(0).(func() gorm.SqlCommon); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(gorm.SqlCommon)
	}

	return r0
}
func (_m *Database) LogMode(enable bool) gorm.Database {
	ret := _m.Called(enable)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(bool) gorm.Database); ok {
		r0 = rf(enable)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) SingularTable(enable bool) {
	_m.Called(enable)
}
func (_m *Database) Where(query interface{}, args ...interface{}) gorm.Database {
	ret := _m.Called(query, args)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(interface{}, ...interface{}) gorm.Database); ok {
		r0 = rf(query, args...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Or(query interface{}, args ...interface{}) gorm.Database {
	ret := _m.Called(query, args)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(interface{}, ...interface{}) gorm.Database); ok {
		r0 = rf(query, args...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Not(query interface{}, args ...interface{}) gorm.Database {
	ret := _m.Called(query, args)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(interface{}, ...interface{}) gorm.Database); ok {
		r0 = rf(query, args...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Limit(value interface{}) gorm.Database {
	ret := _m.Called(value)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(interface{}) gorm.Database); ok {
		r0 = rf(value)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Offset(value interface{}) gorm.Database {
	ret := _m.Called(value)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(interface{}) gorm.Database); ok {
		r0 = rf(value)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Order(value string, reorder ...bool) gorm.Database {
	ret := _m.Called(value, reorder)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(string, ...bool) gorm.Database); ok {
		r0 = rf(value, reorder...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Select(query interface{}, args ...interface{}) gorm.Database {
	ret := _m.Called(query, args)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(interface{}, ...interface{}) gorm.Database); ok {
		r0 = rf(query, args...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Omit(columns ...string) gorm.Database {
	ret := _m.Called(columns)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(...string) gorm.Database); ok {
		r0 = rf(columns...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Group(query string) gorm.Database {
	ret := _m.Called(query)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(string) gorm.Database); ok {
		r0 = rf(query)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Having(query string, values ...interface{}) gorm.Database {
	ret := _m.Called(query, values)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(string, ...interface{}) gorm.Database); ok {
		r0 = rf(query, values...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Joins(query string) gorm.Database {
	ret := _m.Called(query)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(string) gorm.Database); ok {
		r0 = rf(query)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Scopes(funcs ...func(*gorm.DB) *gorm.DB) *gorm.DB {
	ret := _m.Called(funcs)

	var r0 *gorm.DB
	if rf, ok := ret.Get(0).(func(...func(*gorm.DB) *gorm.DB) *gorm.DB); ok {
		r0 = rf(funcs...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*gorm.DB)
		}
	}

	return r0
}
func (_m *Database) Unscoped() gorm.Database {
	ret := _m.Called()

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func() gorm.Database); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Attrs(attrs ...interface{}) gorm.Database {
	ret := _m.Called(attrs)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(...interface{}) gorm.Database); ok {
		r0 = rf(attrs...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Assign(attrs ...interface{}) gorm.Database {
	ret := _m.Called(attrs)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(...interface{}) gorm.Database); ok {
		r0 = rf(attrs...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) First(out interface{}, where ...interface{}) gorm.Database {
	ret := _m.Called(out, where)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(interface{}, ...interface{}) gorm.Database); ok {
		r0 = rf(out, where...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Last(out interface{}, where ...interface{}) gorm.Database {
	ret := _m.Called(out, where)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(interface{}, ...interface{}) gorm.Database); ok {
		r0 = rf(out, where...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Find(out interface{}, where ...interface{}) gorm.Database {
	ret := _m.Called(out, where)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(interface{}, ...interface{}) gorm.Database); ok {
		r0 = rf(out, where...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Scan(dest interface{}) gorm.Database {
	ret := _m.Called(dest)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(interface{}) gorm.Database); ok {
		r0 = rf(dest)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Row() *sql.Row {
	ret := _m.Called()

	var r0 *sql.Row
	if rf, ok := ret.Get(0).(func() *sql.Row); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sql.Row)
		}
	}

	return r0
}
func (_m *Database) Rows() (*sql.Rows, error) {
	ret := _m.Called()

	var r0 *sql.Rows
	if rf, ok := ret.Get(0).(func() *sql.Rows); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sql.Rows)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *Database) Pluck(column string, value interface{}) gorm.Database {
	ret := _m.Called(column, value)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(string, interface{}) gorm.Database); ok {
		r0 = rf(column, value)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Count(value interface{}) gorm.Database {
	ret := _m.Called(value)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(interface{}) gorm.Database); ok {
		r0 = rf(value)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Related(value interface{}, foreignKeys ...string) gorm.Database {
	ret := _m.Called(value, foreignKeys)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(interface{}, ...string) gorm.Database); ok {
		r0 = rf(value, foreignKeys...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) FirstOrInit(out interface{}, where ...interface{}) gorm.Database {
	ret := _m.Called(out, where)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(interface{}, ...interface{}) gorm.Database); ok {
		r0 = rf(out, where...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) FirstOrCreate(out interface{}, where ...interface{}) gorm.Database {
	ret := _m.Called(out, where)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(interface{}, ...interface{}) gorm.Database); ok {
		r0 = rf(out, where...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Update(attrs ...interface{}) gorm.Database {
	ret := _m.Called(attrs)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(...interface{}) gorm.Database); ok {
		r0 = rf(attrs...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Updates(values interface{}, ignoreProtectedAttrs ...bool) gorm.Database {
	ret := _m.Called(values, ignoreProtectedAttrs)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(interface{}, ...bool) gorm.Database); ok {
		r0 = rf(values, ignoreProtectedAttrs...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) UpdateColumn(attrs ...interface{}) gorm.Database {
	ret := _m.Called(attrs)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(...interface{}) gorm.Database); ok {
		r0 = rf(attrs...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) UpdateColumns(values interface{}) gorm.Database {
	ret := _m.Called(values)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(interface{}) gorm.Database); ok {
		r0 = rf(values)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Save(value interface{}) gorm.Database {
	ret := _m.Called(value)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(interface{}) gorm.Database); ok {
		r0 = rf(value)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Create(value interface{}) gorm.Database {
	ret := _m.Called(value)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(interface{}) gorm.Database); ok {
		r0 = rf(value)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Delete(value interface{}, where ...interface{}) gorm.Database {
	ret := _m.Called(value, where)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(interface{}, ...interface{}) gorm.Database); ok {
		r0 = rf(value, where...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Raw(sql string, values ...interface{}) gorm.Database {
	ret := _m.Called(sql, values)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(string, ...interface{}) gorm.Database); ok {
		r0 = rf(sql, values...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Exec(sql string, values ...interface{}) gorm.Database {
	ret := _m.Called(sql, values)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(string, ...interface{}) gorm.Database); ok {
		r0 = rf(sql, values...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Model(value interface{}) gorm.Database {
	ret := _m.Called(value)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(interface{}) gorm.Database); ok {
		r0 = rf(value)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Table(name string) gorm.Database {
	ret := _m.Called(name)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(string) gorm.Database); ok {
		r0 = rf(name)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Debug() gorm.Database {
	ret := _m.Called()

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func() gorm.Database); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Begin() gorm.Database {
	ret := _m.Called()

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func() gorm.Database); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Commit() gorm.Database {
	ret := _m.Called()

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func() gorm.Database); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Rollback() gorm.Database {
	ret := _m.Called()

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func() gorm.Database); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) NewRecord(value interface{}) bool {
	ret := _m.Called(value)

	var r0 bool
	if rf, ok := ret.Get(0).(func(interface{}) bool); ok {
		r0 = rf(value)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}
func (_m *Database) RecordNotFound() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}
func (_m *Database) CreateTable(values ...interface{}) gorm.Database {
	ret := _m.Called(values)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(...interface{}) gorm.Database); ok {
		r0 = rf(values...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) DropTable(values ...interface{}) gorm.Database {
	ret := _m.Called(values)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(...interface{}) gorm.Database); ok {
		r0 = rf(values...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) DropTableIfExists(values ...interface{}) gorm.Database {
	ret := _m.Called(values)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(...interface{}) gorm.Database); ok {
		r0 = rf(values...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) HasTable(value interface{}) bool {
	ret := _m.Called(value)

	var r0 bool
	if rf, ok := ret.Get(0).(func(interface{}) bool); ok {
		r0 = rf(value)
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}
func (_m *Database) AutoMigrate(values ...interface{}) gorm.Database {
	ret := _m.Called(values)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(...interface{}) gorm.Database); ok {
		r0 = rf(values...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) ModifyColumn(column string, typ string) gorm.Database {
	ret := _m.Called(column, typ)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(string, string) gorm.Database); ok {
		r0 = rf(column, typ)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) DropColumn(column string) gorm.Database {
	ret := _m.Called(column)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(string) gorm.Database); ok {
		r0 = rf(column)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) AddIndex(indexName string, column ...string) gorm.Database {
	ret := _m.Called(indexName, column)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(string, ...string) gorm.Database); ok {
		r0 = rf(indexName, column...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) AddUniqueIndex(indexName string, column ...string) gorm.Database {
	ret := _m.Called(indexName, column)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(string, ...string) gorm.Database); ok {
		r0 = rf(indexName, column...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) RemoveIndex(indexName string) gorm.Database {
	ret := _m.Called(indexName)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(string) gorm.Database); ok {
		r0 = rf(indexName)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) CurrentDatabase() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}
func (_m *Database) AddForeignKey(field string, dest string, onDelete string, onUpdate string) gorm.Database {
	ret := _m.Called(field, dest, onDelete, onUpdate)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(string, string, string, string) gorm.Database); ok {
		r0 = rf(field, dest, onDelete, onUpdate)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Association(column string) *gorm.Association {
	ret := _m.Called(column)

	var r0 *gorm.Association
	if rf, ok := ret.Get(0).(func(string) *gorm.Association); ok {
		r0 = rf(column)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*gorm.Association)
		}
	}

	return r0
}
func (_m *Database) Preload(column string, conditions ...interface{}) gorm.Database {
	ret := _m.Called(column, conditions)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(string, ...interface{}) gorm.Database); ok {
		r0 = rf(column, conditions...)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Set(name string, value interface{}) gorm.Database {
	ret := _m.Called(name, value)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(string, interface{}) gorm.Database); ok {
		r0 = rf(name, value)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) InstantSet(name string, value interface{}) gorm.Database {
	ret := _m.Called(name, value)

	var r0 gorm.Database
	if rf, ok := ret.Get(0).(func(string, interface{}) gorm.Database); ok {
		r0 = rf(name, value)
	} else {
		r0 = ret.Get(0).(gorm.Database)
	}

	return r0
}
func (_m *Database) Get(name string) (interface{}, bool) {
	ret := _m.Called(name)

	var r0 interface{}
	if rf, ok := ret.Get(0).(func(string) interface{}); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(interface{})
		}
	}

	var r1 bool
	if rf, ok := ret.Get(1).(func(string) bool); ok {
		r1 = rf(name)
	} else {
		r1 = ret.Get(1).(bool)
	}

	return r0, r1
}
func (_m *Database) SetJoinTableHandler(source interface{}, column string, handler gorm.JoinTableHandlerInterface) {
	_m.Called(source, column, handler)
}
func (_m *Database) AddError(err error) error {
	ret := _m.Called(err)

	var r0 error
	if rf, ok := ret.Get(0).(func(error) error); ok {
		r0 = rf(err)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Database) GetError() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *Database) GetErrors() []error {
	ret := _m.Called()

	var r0 []error
	if rf, ok := ret.Get(0).(func() []error); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]error)
		}
	}

	return r0
}
func (_m *Database) GetRowsAffected() int64 {
	ret := _m.Called()

	var r0 int64
	if rf, ok := ret.Get(0).(func() int64); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(int64)
	}

	return r0
}
func (_m *Database) SetRowsAffected(num int64) {
	_m.Called(num)
}
