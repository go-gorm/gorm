package schema

import (
	"database/sql"
	"go/ast"
	"reflect"
	"time"
)

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

	schema.ModelType = reflectType

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

		schema.Fields = append(schema.Fields, field)
	}

	if len(schema.PrimaryFields) == 0 {
		if field := getSchemaField("id", schema.Fields); field != nil {
			field.IsPrimaryKey = true
			schema.PrimaryFields = append(schema.PrimaryFields, field)
		}
	}

	return &schema
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
