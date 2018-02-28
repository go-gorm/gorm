package model

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/schema"
)

// DefaultTableNameHandler default table name handler
//    DefaultTableNameHandler = func(tx *gorm.DB, tableName string) string {
//    	return tableName
//    }
var DefaultTableNameHandler func(tx *gorm.DB, tableName string) string

// Field GORM model field
type Field struct {
	*schema.Field
	IsBlank bool
	Value   reflect.Value
}

// Set set a value to the field
func (field *Field) Set(value interface{}) (err error) {
	if !field.Value.IsValid() {
		return errors.New("field value not valid")
	}

	if !field.Value.CanAddr() {
		return gorm.ErrUnaddressable
	}

	reflectValue, ok := value.(reflect.Value)
	if !ok {
		reflectValue = reflect.ValueOf(value)
	}

	fieldValue := field.Value
	if reflectValue.IsValid() {
		if reflectValue.Type().ConvertibleTo(fieldValue.Type()) {
			fieldValue.Set(reflectValue.Convert(fieldValue.Type()))
		} else {
			if fieldValue.Kind() == reflect.Ptr {
				if fieldValue.IsNil() {
					fieldValue.Set(reflect.New(field.StructField.Type.Elem()))
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
		field.Value.Set(reflect.Zero(fieldValue.Type()))
	}

	field.IsBlank = IsBlank(field.Value)
	return err
}
