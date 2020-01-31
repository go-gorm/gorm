package schema

import (
	"go/ast"
	"reflect"
	"strings"
	"sync"
)

type Schema struct {
	ModelType               reflect.Type
	Table                   string
	PrioritizedPrimaryField *Field
	PrimaryFields           []*Field
	Fields                  []*Field
	FieldsByName            map[string]*Field
	FieldsByDBName          map[string]*Field
	Relationships           Relationships
	namer                   Namer
}

// get data type from dialector
func Parse(dest interface{}, cacheStore sync.Map, namer Namer) *Schema {
	modelType := reflect.ValueOf(dest).Type()
	for modelType.Kind() == reflect.Slice || modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	if modelType.Kind() != reflect.Struct {
		return nil
	}

	if v, ok := cacheStore.Load(modelType); ok {
		return v.(*Schema)
	}

	schema := &Schema{
		ModelType:      modelType,
		Table:          namer.TableName(modelType.Name()),
		FieldsByName:   map[string]*Field{},
		FieldsByDBName: map[string]*Field{},
	}

	for i := 0; i < modelType.NumField(); i++ {
		fieldStruct := modelType.Field(i)
		if !ast.IsExported(fieldStruct.Name) {
			continue
		}

		field := schema.ParseField(fieldStruct)
		schema.Fields = append(schema.Fields, field)
		if field.EmbeddedbSchema != nil {
			for _, f := range field.EmbeddedbSchema.Fields {
				schema.Fields = append(schema.Fields, f)
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
	}

	return schema
}
