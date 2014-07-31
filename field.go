package gorm

import (
	"database/sql"
	"reflect"
	"time"
)

type relationship struct {
	JoinTable             string
	ForeignKey            string
	AssociationForeignKey string
	Kind                  string
}

type Field struct {
	Name         string
	DBName       string
	Value        interface{}
	Tag          reflect.StructTag
	SqlTag       string
	Relationship *relationship
	IsBlank      bool
	IsIgnored    bool
	IsPrimaryKey bool
}

func (f *Field) IsScanner() bool {
	_, isScanner := reflect.New(reflect.ValueOf(f.Value).Type()).Interface().(sql.Scanner)
	return isScanner
}

func (f *Field) IsTime() bool {
	_, isTime := f.Value.(time.Time)
	return isTime
}
