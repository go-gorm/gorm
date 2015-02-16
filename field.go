package gorm

import (
	"database/sql"
	"errors"
	"reflect"
)

type Field struct {
	*StructField
	IsBlank bool
	Field   reflect.Value
}

func (field *Field) Set(value interface{}) (err error) {
	if !field.Field.IsValid() {
		return errors.New("field value not valid")
	}

	if !field.Field.CanAddr() {
		return errors.New("field value not addressable")
	}

	if rvalue, ok := value.(reflect.Value); ok {
		value = rvalue.Interface()
	}

	if scanner, ok := field.Field.Addr().Interface().(sql.Scanner); ok {
		scanner.Scan(value)
	} else if reflect.TypeOf(value).ConvertibleTo(field.Field.Type()) {
		field.Field.Set(reflect.ValueOf(value).Convert(field.Field.Type()))
	} else {
		return errors.New("could not convert argument")
	}

	field.IsBlank = isBlank(field.Field)

	return
}

type relationship struct {
	JoinTable             string
	ForeignKey            string
	ForeignType           string
	AssociationForeignKey string
	Kind                  string
}

// FIXME
func (r relationship) ForeignDBName() string {
	return ToSnake(r.ForeignKey)
}

func (r relationship) AssociationForeignDBName(name string) string {
	return ToSnake(r.AssociationForeignKey)
}
