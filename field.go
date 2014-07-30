package gorm

import (
	"database/sql"
	"reflect"
	"time"
)

type relationship struct {
	joinTable             string
	foreignKey            string
	associationForeignKey string
}

type Field struct {
	Name              string
	DBName            string
	Value             interface{}
	IsBlank           bool
	IsIgnored         bool
	Tag               reflect.StructTag
	SqlTag            string
	BeforeAssociation bool
	AfterAssociation  bool
	isPrimaryKey      bool
	Relationship      *relationship
}

func (f *Field) IsScanner() bool {
	_, isScanner := reflect.New(reflect.ValueOf(f.Value).Type()).Interface().(sql.Scanner)
	return isScanner
}

func (f *Field) IsTime() bool {
	_, isTime := f.Value.(time.Time)
	return isTime
}
