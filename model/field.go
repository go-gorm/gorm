package model

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/builder"
	"github.com/jinzhu/gorm/schema"
)

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
		return ErrUnaddressable
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

	field.IsBlank = isBlank(field.Value)
	return err
}

// GetAssignments get assignments
func GetAssignments(tx *gorm.DB) chan [][]*Field {
	fieldChan := make(chan [][]*Field)

	go func() {
		assignableChecker := generateAssignableChecker(selectAttrs(tx.Statement), omitAttrs(tx.Statement))

		switch dest := tx.Statement.Dest.(type) {
		case map[string]interface{}:
			fieldChan <- [][]*Field{mapToFields(dest, schema.Parse(tx.Statement.Table), assignableChecker)}
		case []map[string]interface{}:
			fields := [][]*Field{}
			tableSchema := schema.Parse(tx.Statement.Table)

			for _, v := range dest {
				fields = append(fields, mapToFields(v, tableSchema, assignableChecker))
			}
			fieldChan <- fields
		default:
			if s := schema.Parse(tx.Statement.Dest); s != nil {
				results := indirect(reflect.ValueOf(tx.Statement.Dest))

				switch results.Kind() {
				case reflect.Slice:
					fields := [][]*Field{}
					for i := 0; i < results.Len(); i++ {
						fields = append(fields, structToField(indirect(results.Index(i)), s, assignableChecker))
					}
					fieldChan <- fields
				case reflect.Struct:
					fieldChan <- [][]*Field{structToField(results, s, assignableChecker)}
				}
			}
		}
	}()

	return fieldChan
}

func mapToFields(value map[string]interface{}, s *schema.Schema, assignableChecker func(*Field) bool) (fields []*Field) {
	// TODO assign those value to dest
	for k, v := range value {
		if s != nil {
			if f := s.FieldByName(k); f != nil {
				field := &Field{Field: f, Value: reflect.ValueOf(v)}
				if assignableChecker(field) {
					fields = append(fields, field)
				}
				continue
			}
		}

		field := &Field{Field: &schema.Field{DBName: k}, Value: reflect.ValueOf(v)}
		if assignableChecker(field) {
			fields = append(fields, field)
		}
	}

	sort.SliceStable(fields, func(i, j int) bool {
		return strings.Compare(fields[i].Field.DBName, fields[j].Field.DBName) < 0
	})
	return
}

func structToField(value reflect.Value, s *schema.Schema, assignableChecker func(*Field) bool) (fields []*Field) {
	// TODO use Offset to replace FieldByName?
	for _, sf := range s.Fields {
		obj := value
		for _, bn := range sf.BindNames {
			obj = value.FieldByName(bn)
		}
		field := &Field{Field: sf, Value: obj, IsBlank: isBlank(obj)}
		if assignableChecker(field) {
			fields = append(fields, field)
		}
	}
	return
}

// generateAssignableChecker generate checker to check if field is assignable or not
func generateAssignableChecker(selectAttrs []string, omitAttrs []string) func(*Field) bool {
	return func(field *Field) bool {
		if len(selectAttrs) > 0 {
			for _, attr := range selectAttrs {
				if field.Name == attr || field.DBName == attr {
					return true
				}
			}
			return false
		}

		for _, attr := range omitAttrs {
			if field.Name == attr || field.DBName == attr {
				return false
			}
		}
		return true
	}
}

// omitAttrs return selected attributes of stmt
func selectAttrs(stmt *builder.Statement) []string {
	columns := stmt.Select.Columns
	for _, arg := range stmt.Select.Args {
		columns = append(columns, fmt.Sprint(arg))
	}
	return columns
}

// omitAttrs return omitted attributes of stmt
func omitAttrs(stmt *builder.Statement) []string {
	return stmt.Omit
}
