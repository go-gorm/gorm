package schema

import (
	"fmt"
	"go/ast"
	"reflect"
	"strings"
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
	return schema.ModelType.PkgPath()
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
			field := schema.ParseField(fieldStruct)
			schema.Fields = append(schema.Fields, field)
			if field.EmbeddedSchema != nil {
				schema.Fields = append(schema.Fields, field.EmbeddedSchema.Fields...)
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
			}
		}

		if _, ok := schema.FieldsByName[field.Name]; !ok {
			schema.FieldsByName[field.Name] = field
		}
	}

	for db, field := range schema.FieldsByDBName {
		if strings.ToLower(db) == "id" {
			schema.PrioritizedPrimaryField = field
		}

		if field.PrimaryKey {
			if schema.PrioritizedPrimaryField == nil {
				schema.PrioritizedPrimaryField = field
			}
			schema.PrimaryFields = append(schema.PrimaryFields, field)
		}

		if field.DataType == "" {
			defer schema.parseRelation(field)
		}
	}

	cacheStore.Store(modelType, schema)
	return schema, schema.err
}
