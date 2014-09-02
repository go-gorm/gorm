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
	Field        reflect.Value
	Tag          reflect.StructTag
	Relationship *relationship
	IsNormal     bool
	IsBlank      bool
	IsIgnored    bool
	IsPrimaryKey bool
}

func (field *Field) IsScanner() bool {
	_, isScanner := reflect.New(field.Field.Type()).Interface().(sql.Scanner)
	return isScanner
}

func (field *Field) IsTime() bool {
	_, isTime := field.Field.Interface().(time.Time)
	return isTime
}

func (field *Field) Set(value interface{}) (result bool) {
	if field.Field.IsValid() && field.Field.CanAddr() {
		result = true
		if rvalue, ok := value.(reflect.Value); ok {
			value = rvalue.Interface()
		}

		if scanner, ok := field.Field.Addr().Interface().(sql.Scanner); ok {
			scanner.Scan(value)
		} else if reflect.TypeOf(value).ConvertibleTo(field.Field.Type()) {
			field.Field.Set(reflect.ValueOf(value).Convert(field.Field.Type()))
		} else {
			result = false
		}
	}
	if result {
		field.IsBlank = isBlank(field.Field)
	}
	return
}
