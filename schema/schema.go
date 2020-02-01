package schema

import (
	"fmt"
	"go/ast"
	"reflect"
	"sync"

	"github.com/jinzhu/gorm/logger"
)

type Schema struct {
	Name                    string
	ModelType               reflect.Type
	Table                   string
	PrioritizedPrimaryField *Field
	PrimaryFields           []*Field
	Fields                  []*Field
	FieldsByName            map[string]*Field
	FieldsByDBName          map[string]*Field
	Relationships           Relationships
	err                     error
	namer                   Namer
	cacheStore              *sync.Map
}

func (schema Schema) String() string {
	return fmt.Sprintf("%v.%v", schema.ModelType.PkgPath(), schema.ModelType.Name())
}

func (schema Schema) LookUpField(name string) *Field {
	if field, ok := schema.FieldsByDBName[name]; ok {
		return field
	}
	if field, ok := schema.FieldsByName[name]; ok {
		return field
	}
	return nil
}

// get data type from dialector
func Parse(dest interface{}, cacheStore *sync.Map, namer Namer) (*Schema, error) {
	modelType := reflect.ValueOf(dest).Type()
	for modelType.Kind() == reflect.Slice || modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	if modelType.Kind() != reflect.Struct {
		if modelType.PkgPath() == "" {
			return nil, fmt.Errorf("unsupported data %+v when parsing model", dest)
		}
		return nil, fmt.Errorf("unsupported data type %v when parsing model", modelType.PkgPath())
	}

	if v, ok := cacheStore.Load(modelType); ok {
		return v.(*Schema), nil
	}

	schema := &Schema{
		Name:           modelType.Name(),
		ModelType:      modelType,
		Table:          namer.TableName(modelType.Name()),
		FieldsByName:   map[string]*Field{},
		FieldsByDBName: map[string]*Field{},
		Relationships:  Relationships{Relations: map[string]*Relationship{}},
		cacheStore:     cacheStore,
		namer:          namer,
	}

	defer func() {
		if schema.err != nil {
			logger.Default.Error(schema.err.Error())
			cacheStore.Delete(modelType)
		}
	}()

	for i := 0; i < modelType.NumField(); i++ {
		if fieldStruct := modelType.Field(i); ast.IsExported(fieldStruct.Name) {
			if field := schema.ParseField(fieldStruct); field.EmbeddedSchema != nil {
				schema.Fields = append(schema.Fields, field.EmbeddedSchema.Fields...)
			} else {
				schema.Fields = append(schema.Fields, field)
			}
		}
	}

	for _, field := range schema.Fields {
		if field.DBName == "" {
			field.DBName = namer.ColumnName(field.Name)
		}

		if field.DBName != "" {
			// nonexistence or shortest path or first appear prioritized if has permission
			if v, ok := schema.FieldsByDBName[field.DBName]; !ok || (field.Creatable && len(field.BindNames) < len(v.BindNames)) {
				schema.FieldsByDBName[field.DBName] = field
				schema.FieldsByName[field.Name] = field

				if v != nil && v.PrimaryKey {
					if schema.PrioritizedPrimaryField == v {
						schema.PrioritizedPrimaryField = nil
					}

					for idx, f := range schema.PrimaryFields {
						if f == v {
							schema.PrimaryFields = append(schema.PrimaryFields[0:idx], schema.PrimaryFields[idx+1:]...)
						} else if schema.PrioritizedPrimaryField == nil {
							schema.PrioritizedPrimaryField = f
						}
					}
				}

				if field.PrimaryKey {
					if schema.PrioritizedPrimaryField == nil {
						schema.PrioritizedPrimaryField = field
					}
					schema.PrimaryFields = append(schema.PrimaryFields, field)
				}
			}
		}

		if _, ok := schema.FieldsByName[field.Name]; !ok {
			schema.FieldsByName[field.Name] = field
		}
	}

	if f := schema.LookUpField("id"); f != nil {
		if f.PrimaryKey {
			schema.PrioritizedPrimaryField = f
		} else if len(schema.PrimaryFields) == 0 {
			f.PrimaryKey = true
			schema.PrioritizedPrimaryField = f
			schema.PrimaryFields = append(schema.PrimaryFields, f)
		}
	}

	cacheStore.Store(modelType, schema)

	// parse relations for unidentified fields
	for _, field := range schema.Fields {
		if field.DataType == "" && field.Creatable {
			if schema.parseRelation(field); schema.err != nil {
				return schema, schema.err
			}
		}
	}

	return schema, schema.err
}
