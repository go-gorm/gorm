package gorm

import (
	"database/sql"
	"reflect"
	"time"
)

type Field struct {
	Name              string
	DBName            string
	Value             interface{}
	IsBlank           bool
	IsIgnored         bool
	Tag               reflect.StructTag
	SqlTag            string
	ForeignKey        string
	BeforeAssociation bool
	AfterAssociation  bool
	isPrimaryKey      bool
}

func (f *Field) IsScanner() bool {
	_, isScanner := reflect.New(reflect.ValueOf(f.Value).Type()).Interface().(sql.Scanner)
	return isScanner
}

func (f *Field) IsTime() bool {
	_, isTime := f.Value.(time.Time)
	return isTime
}
