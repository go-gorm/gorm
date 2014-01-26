package gorm

import "github.com/jinzhu/gorm/dialect"

type Scope struct {
	Search  *search
	Sql     string
	SqlVars []interface{}
	db      *DB
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
	return true
}

func (scope *Scope) PrimaryKey() string {
	return ""
}

func (scope *Scope) HasColumn(name string) bool {
	return false
}

func (scope *Scope) SetColumn(column string, value interface{}) {
}

func (scope *Scope) CallMethod(name string) {
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
}

func (scope *Scope) Exec() {
}
