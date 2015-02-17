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

// Fields get value's fields
func (scope *Scope) Fields() map[string]*Field {
	if scope.fields == nil {
		fields := map[string]*Field{}
		structFields := scope.GetStructFields()

		for _, structField := range structFields {
			fields[structField.DBName] = scope.getField(structField)
		}

		scope.fields = fields
	}
	return scope.fields
}

func (scope *Scope) getField(structField *StructField) *Field {
	field := Field{StructField: structField}
	indirectValue := scope.IndirectValue()
	if len(structField.Names) > 0 && indirectValue.Kind() == reflect.Struct {
		for _, name := range structField.Names {
			indirectValue = reflect.Indirect(indirectValue).FieldByName(name)
		}
		field.Field = indirectValue
	}
	field.IsBlank = isBlank(indirectValue)
	return &field
}
