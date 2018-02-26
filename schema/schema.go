package schema

import (
	"database/sql"
	"go/ast"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

var schemaMap = sync.Map{}

// Schema model schema definition
type Schema struct {
	ModelType     reflect.Type
	PrimaryFields []*Field
	Fields        []*Field
	TableName     string
	ParseErrors   []error
}

// Field schema field definition
type Field struct {
	DBName          string
	Name            string
	BindNames       []string
	Tag             reflect.StructTag
	TagSettings     map[string]string
	IsNormal        bool
	IsPrimaryKey    bool
	IsIgnored       bool
	IsForeignKey    bool
	DefaultValue    string
	HasDefaultValue bool
	StructField     reflect.StructField
	Relationship    *Relationship
}

// Parse parse struct and generate schema based on struct and tag definition
func Parse(dest interface{}) *Schema {
	schema := Schema{}

	// Get dest type
	reflectType := reflect.ValueOf(dest).Type()
	for reflectType.Kind() == reflect.Slice || reflectType.Kind() == reflect.Ptr {
		reflectType = reflectType.Elem()
	}

	if reflectType.Kind() != reflect.Struct {
		return nil
	}

	if m, ok := schemaMap.Load(reflectType); ok {
		return m.(*Schema)
	}

	schema.ModelType = reflectType
	onConflictFields := map[string]int{}

	for i := 0; i < reflectType.NumField(); i++ {
		fieldStruct := reflectType.Field(i)
		if !ast.IsExported(fieldStruct.Name) {
			continue
		}

		field := &Field{
			Name:        fieldStruct.Name,
			BindNames:   []string{fieldStruct.Name},
			StructField: fieldStruct,
			Tag:         fieldStruct.Tag,
			TagSettings: parseTagSetting(fieldStruct.Tag),
		}

		if _, ok := field.TagSettings["-"]; ok {
			field.IsIgnored = true
		} else {
			if val, ok := field.TagSettings["PRIMARY_KEY"]; ok && checkTruth(val) {
				field.IsPrimaryKey = true
				schema.PrimaryFields = append(schema.PrimaryFields, field)
			}

			if val, ok := field.TagSettings["AUTO_INCREMENT"]; ok && checkTruth(val) && !field.IsPrimaryKey {
				field.HasDefaultValue = true
			}

			if v, ok := field.TagSettings["DEFAULT"]; ok {
				field.DefaultValue = v
				field.HasDefaultValue = true
			}

			indirectType := fieldStruct.Type
			for indirectType.Kind() == reflect.Ptr {
				indirectType = indirectType.Elem()
			}

			fieldValue := reflect.New(indirectType).Interface()
			if _, isScanner := fieldValue.(sql.Scanner); isScanner {
				// scanner
				field.IsNormal = true
				if indirectType.Kind() == reflect.Struct {
					// Use tag settings from scanner
					for i := 0; i < indirectType.NumField(); i++ {
						for key, value := range parseTagSetting(indirectType.Field(i).Tag) {
							if _, ok := field.TagSettings[key]; !ok {
								field.TagSettings[key] = value
							}
						}
					}
				}
			} else if _, isTime := fieldValue.(*time.Time); isTime {
				// time
				field.IsNormal = true
			} else if _, ok := field.TagSettings["EMBEDDED"]; ok || fieldStruct.Anonymous {
				// embedded struct
				if subSchema := Parse(fieldValue); subSchema != nil {
					for _, subField := range subSchema.Fields {
						subField = subField.clone()
						subField.BindNames = append([]string{fieldStruct.Name}, subField.BindNames...)
						if prefix, ok := field.TagSettings["EMBEDDED_PREFIX"]; ok {
							subField.DBName = prefix + subField.DBName
						}

						if subField.IsPrimaryKey {
							if _, ok := subField.TagSettings["PRIMARY_KEY"]; ok {
								schema.PrimaryFields = append(schema.PrimaryFields, subField)
							} else {
								subField.IsPrimaryKey = false
							}
						}

						for key, value := range field.TagSettings {
							subField.TagSettings[key] = value
						}

						schema.Fields = append(schema.Fields, subField)
					}
				}
				continue
			} else {
				// build relationships
				switch indirectType.Kind() {
				case reflect.Struct:
					defer buildToOneRel(field, &schema)
				case reflect.Slice:
					defer buildToManyRel(field, &schema)
				default:
					field.IsNormal = true
				}
			}
		}

		// Even it is ignored, also possible to decode db value into the field
		if dbName, ok := field.TagSettings["COLUMN"]; ok {
			field.DBName = dbName
		} else if field.DBName == "" {
			field.DBName = ToDBName(fieldStruct.Name)
		}

		if _, ok := field.TagSettings["ON_EMBEDDED_CONFLICT"]; ok {
			onConflictFields[field.Name] = len(schema.Fields)
		}

		schema.Fields = append(schema.Fields, field)
	}

	if len(onConflictFields) > 0 {
		removeIdx := []int{}

		updatePrimaryKey := func(field, conflictField *Field) {
			if field != nil && field.IsPrimaryKey {
				for i, p := range schema.PrimaryFields {
					if p == field {
						schema.PrimaryFields = append(schema.PrimaryFields[0:i], schema.PrimaryFields[i+1:]...)
					}
				}
			}

			if conflictField != nil && conflictField.IsPrimaryKey {
				schema.PrimaryFields = append(schema.PrimaryFields, conflictField)
			}
		}

		for _, idx := range onConflictFields {
			conflictField := schema.Fields[idx]

			for i, field := range schema.Fields {
				if i != idx && conflictField.Name == field.Name {
					switch conflictField.TagSettings["ON_EMBEDDED_CONFLICT"] {
					case "replace":
						// if original field is primary key, delete origianl one
						// add conflicated one if it is primary key
						if field.IsPrimaryKey {
							updatePrimaryKey(field, conflictField)
						}
						removeIdx = append(removeIdx, i)
					case "ignore":
						// skip ignored field
						updatePrimaryKey(conflictField, nil)
						removeIdx = append(removeIdx, idx)
					case "update":
						// if original field is primary key, delete origianl one
						// add conflicated one if it is primary key
						if field.IsPrimaryKey {
							updatePrimaryKey(field, conflictField)
						}
						for key, value := range field.TagSettings {
							if _, ok := conflictField.TagSettings[key]; !ok {
								conflictField.TagSettings[key] = value
							}
						}

						conflictField.BindNames = field.BindNames
						if column, ok := conflictField.TagSettings["COLUMN"]; ok {
							conflictField.DBName = column
						}
						*field = *conflictField
						removeIdx = append(removeIdx, idx)
					}
				}
			}
		}

		sort.Ints(removeIdx)
		for i := len(removeIdx) - 1; i >= 0; i-- {
			schema.Fields = append(schema.Fields[0:removeIdx[i]], schema.Fields[removeIdx[i]+1:]...)
		}
	}

	if len(schema.PrimaryFields) == 0 {
		for _, field := range schema.Fields {
			if strings.ToUpper(field.Name) == "ID" || field.DBName == "id" {
				field.IsPrimaryKey = true
				schema.PrimaryFields = append(schema.PrimaryFields, field)
				break
			}
		}
	}

	schemaMap.Store(reflectType, &schema)
	return &schema
}

// MainPrimaryField returns main primary field, usually the field with db name "id" or the first primary field
func (schema *Schema) MainPrimaryField() *Field {
	for _, field := range schema.PrimaryFields {
		if field.DBName == "id" {
			return field
		}
	}
	if len(schema.PrimaryFields) > 0 {
		return schema.PrimaryFields[0]
	}
	return nil
}

func (schemaField *Field) clone() *Field {
	clone := *schemaField

	if schemaField.Relationship != nil {
		relationship := *schemaField.Relationship
		clone.Relationship = &relationship
	}

	for key, value := range schemaField.TagSettings {
		clone.TagSettings[key] = value
	}

	return &clone
}
