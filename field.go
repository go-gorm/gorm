package gorm

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
)

type Field struct {
	*StructField
	IsBlank bool
	Field   reflect.Value
}

func (field *Field) Set(value interface{}) error {
	if !field.Field.IsValid() {
		return errors.New("field value not valid")
	}

	if !field.Field.CanAddr() {
		return errors.New("unaddressable value")
	}

	if rvalue, ok := value.(reflect.Value); ok {
		value = rvalue.Interface()
	}

	if scanner, ok := field.Field.Addr().Interface().(sql.Scanner); ok {
		if v, ok := value.(reflect.Value); ok {
			if err := scanner.Scan(v.Interface()); err != nil {
				return err
			}
		} else {
			if err := scanner.Scan(value); err != nil {
				return err
			}
		}
	} else {
		reflectValue, ok := value.(reflect.Value)
		if !ok {
			reflectValue = reflect.ValueOf(value)
		}

		if !reflectValue.IsValid() {
			return nil
		}

		if reflectValue.Type().ConvertibleTo(field.Field.Type()) {
			field.Field.Set(reflectValue.Convert(field.Field.Type()))
		} else {
			return fmt.Errorf("could not convert argument of field %s from %s to %s", field.Name, reflectValue.Type(), field.Field.Type())
		}
	}

	field.IsBlank = isBlank(field.Field)
	return nil
}

// Fields get value's fields
func (scope *Scope) Fields() map[string]*Field {
	if scope.fields == nil {
		fields := map[string]*Field{}
		modelStruct := scope.GetModelStruct()

		indirectValue := scope.IndirectValue()
		isStruct := indirectValue.Kind() == reflect.Struct
		for _, structField := range modelStruct.StructFields {
			if field, ok := fields[structField.DBName]; !ok || field.IsIgnored {
				if isStruct {
					fields[structField.DBName] = getField(indirectValue, structField)
				} else {
					fields[structField.DBName] = &Field{StructField: structField, IsBlank: true}
				}
			}
		}

		if modelStruct.cached {
			scope.fields = fields
		}
		return fields
	}
	return scope.fields
}

func getField(indirectValue reflect.Value, structField *StructField) *Field {
	field := &Field{StructField: structField}
	for _, name := range structField.Names {
		indirectValue = reflect.Indirect(indirectValue).FieldByName(name)
	}
	field.Field = indirectValue
	field.IsBlank = isBlank(indirectValue)
	return field
}
