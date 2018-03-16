package sqlbuilder

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/model"
	"github.com/jinzhu/gorm/schema"
)

// GetAssignmentFields get assignment fields
func GetAssignmentFields(tx *gorm.DB) chan [][]*model.Field {
	fieldChan := make(chan [][]*model.Field)

	go func() {
		assignableChecker := generateAssignableChecker(selectAttrs(tx.Statement), omitAttrs(tx.Statement))

		switch dest := tx.Statement.Dest.(type) {
		case map[string]interface{}:
			fieldChan <- [][]*model.Field{mapToFields(dest, schema.Parse(tx.Statement.Table), assignableChecker)}
		case []map[string]interface{}:
			fields := [][]*model.Field{}
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
					fields := [][]*model.Field{}
					for i := 0; i < results.Len(); i++ {
						fields = append(fields, structToField(indirect(results.Index(i)), s, assignableChecker))
					}
					fieldChan <- fields
				case reflect.Struct:
					fieldChan <- [][]*model.Field{structToField(results, s, assignableChecker)}
				}
			}
		}
	}()

	return fieldChan
}

// GetSelectableFields get selectable fields
func GetSelectableFields(tx *gorm.DB) chan []string {
	fieldChan := make(chan []string)

	go func() {
		assignableChecker := generateAssignableChecker(selectAttrs(tx.Statement), omitAttrs(tx.Statement))
		if s := schema.Parse(tx.Statement.Dest); s != nil {
			columns := []string{}
			for _, field := range s.Fields {
				if assignableChecker(field) {
					columns = append(columns, field.DBName)
				}
			}
			fieldChan <- columns
			return
		}
		fieldChan <- []string{"*"}
	}()

	return fieldChan
}

func mapToFields(value map[string]interface{}, s *schema.Schema, assignableChecker func(*schema.Field) bool) (fields []*model.Field) {
	// TODO assign those value to dest
	for k, v := range value {
		if s != nil {
			if f := s.FieldByName(k); f != nil {
				field := &model.Field{Field: f, Value: reflect.ValueOf(v)}
				if assignableChecker(field.Field) {
					fields = append(fields, field)
				}
				continue
			}
		}

		field := &model.Field{Field: &schema.Field{DBName: k}, Value: reflect.ValueOf(v)}
		if assignableChecker(field.Field) {
			fields = append(fields, field)
		}
	}

	sort.SliceStable(fields, func(i, j int) bool {
		return strings.Compare(fields[i].Field.DBName, fields[j].Field.DBName) < 0
	})
	return
}

func structToField(value reflect.Value, s *schema.Schema, assignableChecker func(*schema.Field) bool) (fields []*model.Field) {
	// TODO use Offset to replace FieldByName?
	for _, sf := range s.Fields {
		obj := value
		for _, bn := range sf.BindNames {
			obj = value.FieldByName(bn)
		}
		field := &model.Field{Field: sf, Value: obj, IsBlank: model.IsBlank(obj)}
		if assignableChecker(field.Field) {
			fields = append(fields, field)
		}
	}
	return
}

// generateAssignableChecker generate checker to check if field is assignable or not
func generateAssignableChecker(selectAttrs []string, omitAttrs []string) func(*schema.Field) bool {
	return func(field *schema.Field) bool {
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
func selectAttrs(stmt *gorm.Statement) []string {
	columns := stmt.Select.Columns
	for _, arg := range stmt.Select.Args {
		columns = append(columns, fmt.Sprint(arg))
	}
	return columns
}

// omitAttrs return omitted attributes of stmt
func omitAttrs(stmt *gorm.Statement) []string {
	return stmt.Omit
}
