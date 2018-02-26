package model

import (
	"reflect"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/schema"
)

// Field GORM model field
type Field struct {
	*schema.Field
	IsBlank bool
	Value   reflect.Value
}

// GetCreatingAssignments get creating assignments
func GetCreatingAssignments(tx *gorm.DB) chan [][]*Field {
	fieldChan := make(chan [][]*Field)

	go func() {
		// TODO handle select, omit, protected
		switch dest := tx.Statement.Dest.(type) {
		case map[string]interface{}:
			fieldChan <- [][]*Field{mapToFields(dest, schema.Parse(tx.Statement.Table))}
		case []map[string]interface{}:
			fields := [][]*Field{}
			tableSchema := schema.Parse(tx.Statement.Table)

			for _, v := range dest {
				fields = append(fields, mapToFields(v, tableSchema))
			}
			fieldChan <- fields
		default:
			if s := schema.Parse(tx.Statement.Dest); s != nil {
				results := indirect(reflect.ValueOf(tx.Statement.Dest))

				switch results.Kind() {
				case reflect.Slice:
					fields := [][]*Field{}
					for i := 0; i < results.Len(); i++ {
						fields = append(fields, structToField(results.Index(i), s))
					}
					fieldChan <- fields
				case reflect.Struct:
					fieldChan <- [][]*Field{structToField(results, s)}
				}
			}
		}
	}()

	return fieldChan
}

func mapToFields(value map[string]interface{}, s *schema.Schema) (fields []*Field) {
	for k, v := range value {
		if s != nil {
			if f := s.FieldByName(k); f != nil {
				fields = append(fields, &Field{Field: f, Value: reflect.ValueOf(v)})
				continue
			}
		}

		fields = append(fields, &Field{Field: &schema.Field{DBName: k}, Value: reflect.ValueOf(v)})
	}
	return
}

func structToField(value reflect.Value, s *schema.Schema) (fields []*Field) {
	for _, sf := range s.Fields {
		obj := value
		for _, bn := range sf.BindNames {
			obj = value.FieldByName(bn)
		}
		fields = append(fields, &Field{Field: sf, Value: obj})
	}
	return
}
