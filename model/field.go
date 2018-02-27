package model

import (
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

// GetAssignments get assignments
func GetAssignments(tx *gorm.DB) chan [][]*Field {
	fieldChan := make(chan [][]*Field)

	go func() {
		// TODO handle select, omit, protected
		switch dest := tx.Statement.Dest.(type) {
		case map[string]interface{}:
			fieldChan <- [][]*Field{mapToFields(dest, tx.Statement, schema.Parse(tx.Statement.Table))}
		case []map[string]interface{}:
			fields := [][]*Field{}
			tableSchema := schema.Parse(tx.Statement.Table)

			for _, v := range dest {
				fields = append(fields, mapToFields(v, tx.Statement, tableSchema))
			}
			fieldChan <- fields
		default:
			if s := schema.Parse(tx.Statement.Dest); s != nil {
				results := indirect(reflect.ValueOf(tx.Statement.Dest))

				switch results.Kind() {
				case reflect.Slice:
					fields := [][]*Field{}
					for i := 0; i < results.Len(); i++ {
						fields = append(fields, structToField(results.Index(i), tx.Statement, s))
					}
					fieldChan <- fields
				case reflect.Struct:
					fieldChan <- [][]*Field{structToField(results, tx.Statement, s)}
				}
			}
		}
	}()

	return fieldChan
}

func mapToFields(value map[string]interface{}, stmt *builder.Statement, s *schema.Schema) (fields []*Field) {
	// sort
	// TODO assign those value to dest
	for k, v := range value {
		if s != nil {
			if f := s.FieldByName(k); f != nil {
				fields = append(fields, &Field{Field: f, Value: reflect.ValueOf(v)})
				continue
			}
		}

		fields = append(fields, &Field{Field: &schema.Field{DBName: k}, Value: reflect.ValueOf(v)})
	}

	sort.SliceStable(fields, func(i, j int) bool {
		return strings.Compare(fields[i].Field.DBName, fields[j].Field.DBName) < 0
	})
	return
}

func structToField(value reflect.Value, stmt *builder.Statement, s *schema.Schema) (fields []*Field) {
	// TODO use Offset to replace FieldByName?
	for _, sf := range s.Fields {
		obj := value
		for _, bn := range sf.BindNames {
			obj = value.FieldByName(bn)
		}
		fields = append(fields, &Field{Field: sf, Value: obj, IsBlank: isBlank(obj)})
	}
	return
}
