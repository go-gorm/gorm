package gorm

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
)

// Field model field definition
type Field struct {
	*StructField
	IsBlank bool
	Field   reflect.Value
}

func (field *Field) CallMethodCallbackArgs(name string, object reflect.Value, in []reflect.Value) {
	field.StructField.CallMethodCallbackArgs(name, object, append([]reflect.Value{reflect.ValueOf(field)}, in...))
}

// Call the method callback
func (field *Field) CallMethodCallback(name string, object reflect.Value, in ...reflect.Value) {
	field.CallMethodCallbackArgs(name, object, in)
}

// Set set a value to the field
func (field *Field) Set(value interface{}) (err error) {
	if !field.Field.IsValid() {
		return errors.New("field value not valid")
	}

	if !field.Field.CanAddr() {
		return ErrUnaddressable
	}

	reflectValue, ok := value.(reflect.Value)
	if !ok {
		reflectValue = reflect.ValueOf(value)
	}

	fieldValue := field.Field
	if reflectValue.IsValid() {
		if reflectValue.Type().ConvertibleTo(fieldValue.Type()) {
			fieldValue.Set(reflectValue.Convert(fieldValue.Type()))
		} else {
			if fieldValue.Kind() == reflect.Ptr {
				if fieldValue.IsNil() {
					fieldValue.Set(reflect.New(field.Struct.Type.Elem()))
				}
				fieldValue = fieldValue.Elem()
			}

			if reflectValue.Type().ConvertibleTo(fieldValue.Type()) {
				fieldValue.Set(reflectValue.Convert(fieldValue.Type()))
			} else if scanner, ok := fieldValue.Addr().Interface().(sql.Scanner); ok {
				err = scanner.Scan(reflectValue.Interface())
			} else {
				err = fmt.Errorf("could not convert argument of field %s from %s to %s", field.Name, reflectValue.Type(), fieldValue.Type())
			}
		}
	} else {
		field.Field.Set(reflect.Zero(field.Field.Type()))
	}

	field.IsBlank = isBlank(field.Field)
	return err
}
