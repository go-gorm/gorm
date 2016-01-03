package gorm

import (
	"database/sql"
	"fmt"
	"go/ast"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jinzhu/inflection"
)

var DefaultTableNameHandler = func(db *DB, defaultTableName string) string {
	return defaultTableName
}

type safeModelStructsMap struct {
	m map[reflect.Type]*ModelStruct
	l *sync.RWMutex
}

func (s *safeModelStructsMap) Set(key reflect.Type, value *ModelStruct) {
	s.l.Lock()
	defer s.l.Unlock()
	s.m[key] = value
}

func (s *safeModelStructsMap) Get(key reflect.Type) *ModelStruct {
	s.l.RLock()
	defer s.l.RUnlock()
	return s.m[key]
}

func newModelStructsMap() *safeModelStructsMap {
	return &safeModelStructsMap{l: new(sync.RWMutex), m: make(map[reflect.Type]*ModelStruct)}
}

var modelStructsMap = newModelStructsMap()

type ModelStruct struct {
	PrimaryFields    []*StructField
	StructFields     []*StructField
	ModelType        reflect.Type
	defaultTableName string
}

func (s *ModelStruct) TableName(db *DB) string {
	return DefaultTableNameHandler(db, s.defaultTableName)
}

type StructField struct {
	DBName          string
	Name            string
	Names           []string
	IsPrimaryKey    bool
	IsNormal        bool
	IsIgnored       bool
	IsScanner       bool
	HasDefaultValue bool
	Tag             reflect.StructTag
	TagSettings     map[string]string
	Struct          reflect.StructField
	IsForeignKey    bool
	Relationship    *Relationship
}

func (structField *StructField) clone() *StructField {
	return &StructField{
		DBName:          structField.DBName,
		Name:            structField.Name,
		Names:           structField.Names,
		IsPrimaryKey:    structField.IsPrimaryKey,
		IsNormal:        structField.IsNormal,
		IsIgnored:       structField.IsIgnored,
		IsScanner:       structField.IsScanner,
		HasDefaultValue: structField.HasDefaultValue,
		Tag:             structField.Tag,
		Struct:          structField.Struct,
		IsForeignKey:    structField.IsForeignKey,
		Relationship:    structField.Relationship,
	}
}

type Relationship struct {
	Kind                         string
	PolymorphicType              string
	PolymorphicDBName            string
	ForeignFieldNames            []string
	ForeignDBNames               []string
	AssociationForeignFieldNames []string
	AssociationForeignDBNames    []string
	JoinTableHandler             JoinTableHandlerInterface
}

func getForeignField(column string, fields []*StructField) *StructField {
	for _, field := range fields {
		if field.Name == column || field.DBName == ToDBName(column) {
			return field
		}
	}
	return nil
}

func (scope *Scope) GetModelStruct() *ModelStruct {
	var modelStruct ModelStruct
	// Scope value can't be nil
	if scope.Value == nil {
		return &modelStruct
	}

	reflectType := reflect.ValueOf(scope.Value).Type()
	for reflectType.Kind() == reflect.Slice || reflectType.Kind() == reflect.Ptr {
		reflectType = reflectType.Elem()
	}

	// Scope value need to be a struct
	if reflectType.Kind() != reflect.Struct {
		return &modelStruct
	}

	// Get Cached model struct
	if value := modelStructsMap.Get(reflectType); value != nil {
		return value
	}

	modelStruct.ModelType = reflectType

	// Set default table name
	if tabler, ok := reflect.New(reflectType).Interface().(tabler); ok {
		modelStruct.defaultTableName = tabler.TableName()
	} else {
		tableName := ToDBName(reflectType.Name())
		if scope.db == nil || !scope.db.parent.singularTable {
			tableName = inflection.Plural(tableName)
		}
		modelStruct.defaultTableName = tableName
	}

	// Get all fields
	for i := 0; i < reflectType.NumField(); i++ {
		if fieldStruct := reflectType.Field(i); ast.IsExported(fieldStruct.Name) {
			field := &StructField{
				Struct:      fieldStruct,
				Name:        fieldStruct.Name,
				Names:       []string{fieldStruct.Name},
				Tag:         fieldStruct.Tag,
				TagSettings: parseTagSetting(fieldStruct.Tag),
			}

			if fieldStruct.Tag.Get("sql") == "-" {
				field.IsIgnored = true
			} else {
				if _, ok := field.TagSettings["PRIMARY_KEY"]; ok {
					field.IsPrimaryKey = true
					modelStruct.PrimaryFields = append(modelStruct.PrimaryFields, field)
				}

				if _, ok := field.TagSettings["DEFAULT"]; ok {
					field.HasDefaultValue = true
				}

				fieldValue := reflect.New(fieldStruct.Type).Interface()
				if _, isScanner := fieldValue.(sql.Scanner); isScanner {
					// is scanner
					field.IsScanner, field.IsNormal = true, true
				} else if _, isTime := fieldValue.(*time.Time); isTime {
					// is time
					field.IsNormal = true
				} else if _, ok := field.TagSettings["EMBEDDED"]; ok || fieldStruct.Anonymous {
					// embedded struct
					for _, subField := range scope.New(fieldValue).GetStructFields() {
						subField = subField.clone()
						subField.Names = append([]string{fieldStruct.Name}, subField.Names...)
						if subField.IsPrimaryKey {
							modelStruct.PrimaryFields = append(modelStruct.PrimaryFields, subField)
						}
						modelStruct.StructFields = append(modelStruct.StructFields, subField)
					}
					continue
				} else {
					indirectType := fieldStruct.Type
					for indirectType.Kind() == reflect.Ptr {
						indirectType = indirectType.Elem()
					}

					switch indirectType.Kind() {
					case reflect.Slice:
						defer func(field *StructField) {
							var (
								relationship           = &Relationship{}
								toScope                = scope.New(reflect.New(field.Struct.Type).Interface())
								foreignKeys            []string
								associationForeignKeys []string
								elemType               = field.Struct.Type
							)

							if foreignKey := field.TagSettings["FOREIGNKEY"]; foreignKey != "" {
								foreignKeys = strings.Split(field.TagSettings["FOREIGNKEY"], ";")
							}

							if foreignKey := field.TagSettings["ASSOCIATIONFOREIGNKEY"]; foreignKey != "" {
								associationForeignKeys = strings.Split(field.TagSettings["ASSOCIATIONFOREIGNKEY"], ";")
							}

							for elemType.Kind() == reflect.Slice || elemType.Kind() == reflect.Ptr {
								elemType = elemType.Elem()
							}

							if elemType.Kind() == reflect.Struct {
								if many2many := field.TagSettings["MANY2MANY"]; many2many != "" {
									relationship.Kind = "many_to_many"

									// if no foreign keys
									if len(foreignKeys) == 0 {
										for _, field := range scope.PrimaryFields() { // FIXME
											foreignKeys = append(foreignKeys, field.DBName)
										}
									}

									for _, foreignKey := range foreignKeys {
										if field, ok := scope.FieldByName(foreignKey); ok { // FIXME
											// source foreign keys (db names)
											relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, field.DBName)
											// join table foreign keys for source
											joinTableDBName := ToDBName(reflectType.Name()) + "_" + field.DBName
											relationship.ForeignDBNames = append(relationship.ForeignDBNames, joinTableDBName)
										}
									}

									// if no association foreign keys
									if len(associationForeignKeys) == 0 {
										for _, field := range toScope.PrimaryFields() {
											associationForeignKeys = append(associationForeignKeys, field.DBName)
										}
									}

									for _, name := range associationForeignKeys {
										if field, ok := toScope.FieldByName(name); ok {
											// association foreign keys (db names)
											relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, field.DBName)
											// join table foreign keys for association
											joinTableDBName := ToDBName(elemType.Name()) + "_" + field.DBName
											relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, joinTableDBName)
										}
									}

									joinTableHandler := JoinTableHandler{}
									joinTableHandler.Setup(relationship, many2many, reflectType, elemType)
									relationship.JoinTableHandler = &joinTableHandler
									field.Relationship = relationship
								} else {
									relationship.Kind = "has_many"

									// if no foreign keys
									if len(foreignKeys) == 0 {
										if len(associationForeignKeys) == 0 {
											for _, field := range modelStruct.PrimaryFields {
												foreignKeys = append(foreignKeys, reflectType.Name()+field.Name)
												associationForeignKeys = append(associationForeignKeys, field.Name)
											}
										} else {
											for _, associationForeignKey := range associationForeignKeys {
												if foreignField := getForeignField(associationForeignKey, modelStruct.StructFields); foreignField != nil {
													foreignKeys = append(foreignKeys, reflectType.Name()+foreignField.Name)
													associationForeignKeys = append(associationForeignKeys, foreignField.Name)
												}
											}
										}
									}

									for idx, foreignKey := range foreignKeys {
										if foreignField := getForeignField(foreignKey, toScope.GetStructFields()); foreignField != nil {
											if associationField := getForeignField(associationForeignKeys[idx], modelStruct.StructFields); associationField != nil {
												// source foreign keys
												foreignField.IsForeignKey = true
												relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, associationField.Name)
												relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, associationField.DBName)

												// association foreign keys
												relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, foreignField.Name)
												relationship.ForeignDBNames = append(relationship.ForeignDBNames, foreignField.DBName)
											}
										}
									}

									if len(relationship.ForeignFieldNames) != 0 {
										field.Relationship = relationship
									}
								}
							} else {
								field.IsNormal = true
							}
						}(field)
					case reflect.Struct:
						defer func(field *StructField) {
							var (
								relationship              = &Relationship{}
								toScope                   = scope.New(reflect.New(field.Struct.Type).Interface())
								tagForeignKeys            []string
								tagAssociationForeignKeys []string
							)

							if foreignKey := field.TagSettings["FOREIGNKEY"]; foreignKey != "" {
								tagForeignKeys = strings.Split(field.TagSettings["FOREIGNKEY"], ";")
							}

							if foreignKey := field.TagSettings["ASSOCIATIONFOREIGNKEY"]; foreignKey != "" {
								tagAssociationForeignKeys = strings.Split(field.TagSettings["ASSOCIATIONFOREIGNKEY"], ";")
							}

							// Has One
							{
								var foreignKeys = tagForeignKeys
								var associationForeignKeys = tagAssociationForeignKeys
								if len(foreignKeys) == 0 {
									if len(associationForeignKeys) == 0 {
										for _, primaryField := range modelStruct.PrimaryFields {
											foreignKeys = append(foreignKeys, modelStruct.ModelType.Name()+primaryField.Name)
											associationForeignKeys = append(associationForeignKeys, primaryField.Name)
										}
									} else {
										for _, associationForeignKey := range tagAssociationForeignKeys {
											if foreignField := getForeignField(associationForeignKey, modelStruct.StructFields); foreignField != nil {
												foreignKeys = append(foreignKeys, modelStruct.ModelType.Name()+foreignField.Name)
												associationForeignKeys = append(associationForeignKeys, foreignField.Name)
											}
										}
									}
								}

								for idx, foreignKey := range foreignKeys {
									if foreignField := getForeignField(foreignKey, toScope.GetStructFields()); foreignField != nil {
										if scopeField := getForeignField(associationForeignKeys[idx], modelStruct.StructFields); scopeField != nil {
											foreignField.IsForeignKey = true
											// source foreign keys
											relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, scopeField.Name)
											relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, scopeField.DBName)

											// association foreign keys
											relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, foreignField.Name)
											relationship.ForeignDBNames = append(relationship.ForeignDBNames, foreignField.DBName)
										}
									}
								}
							}

							if len(relationship.ForeignFieldNames) != 0 {
								relationship.Kind = "has_one"
								field.Relationship = relationship
							} else {
								var foreignKeys = tagForeignKeys
								var associationForeignKeys = tagAssociationForeignKeys
								if len(foreignKeys) == 0 {
									if len(associationForeignKeys) == 0 {
										for _, f := range toScope.PrimaryFields() {
											foreignKeys = append(foreignKeys, field.Name+f.Name)
											associationForeignKeys = append(associationForeignKeys, f.Name)
										}
									} else {
										for _, associationForeignKey := range associationForeignKeys {
											if foreignField := getForeignField(associationForeignKey, toScope.GetStructFields()); foreignField != nil {
												foreignKeys = append(foreignKeys, field.Name+foreignField.Name)
												associationForeignKeys = append(associationForeignKeys, foreignField.Name)
											}
										}
									}
								}

								for idx, foreignKey := range foreignKeys {
									if foreignField := getForeignField(foreignKey, modelStruct.StructFields); foreignField != nil {
										if associationField := getForeignField(associationForeignKeys[idx], toScope.GetStructFields()); associationField != nil {
											// association foreign keys
											relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, associationField.Name)
											relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, associationField.DBName)

											// source foreign keys
											relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, foreignField.Name)
											relationship.ForeignDBNames = append(relationship.ForeignDBNames, foreignField.DBName)
											foreignField.IsForeignKey = true
										}
									}
								}

								if len(relationship.ForeignFieldNames) != 0 {
									relationship.Kind = "belongs_to"
									field.Relationship = relationship
								}
							}
						}(field)
					default:
						field.IsNormal = true
					}
				}
			}

			// Even it is ignored, also possible to decode db value into the field
			if value, ok := field.TagSettings["COLUMN"]; ok {
				field.DBName = value
			} else {
				field.DBName = ToDBName(fieldStruct.Name)
			}

			modelStruct.StructFields = append(modelStruct.StructFields, field)
		}
	}

	if len(modelStruct.PrimaryFields) == 0 {
		if field := getForeignField("id", modelStruct.StructFields); field != nil {
			field.IsPrimaryKey = true
			modelStruct.PrimaryFields = append(modelStruct.PrimaryFields, field)
		}
	}

	modelStructsMap.Set(reflectType, &modelStruct)

	return &modelStruct
}

func (scope *Scope) GetStructFields() (fields []*StructField) {
	return scope.GetModelStruct().StructFields
}

func (scope *Scope) generateSqlTag(field *StructField) string {
	var sqlType string
	structType := field.Struct.Type
	if structType.Kind() == reflect.Ptr {
		structType = structType.Elem()
	}
	reflectValue := reflect.Indirect(reflect.New(structType))

	if value, ok := field.TagSettings["TYPE"]; ok {
		sqlType = value
	}

	additionalType := field.TagSettings["NOT NULL"] + " " + field.TagSettings["UNIQUE"]
	if value, ok := field.TagSettings["DEFAULT"]; ok {
		additionalType = additionalType + " DEFAULT " + value
	}

	if field.IsScanner {
		var getScannerValue func(reflect.Value)
		getScannerValue = func(value reflect.Value) {
			reflectValue = value
			if _, isScanner := reflect.New(reflectValue.Type()).Interface().(sql.Scanner); isScanner && reflectValue.Kind() == reflect.Struct {
				getScannerValue(reflectValue.Field(0))
			}
		}
		getScannerValue(reflectValue)
	}

	if sqlType == "" {
		var size = 255

		if value, ok := field.TagSettings["SIZE"]; ok {
			size, _ = strconv.Atoi(value)
		}

		v, autoIncrease := field.TagSettings["AUTO_INCREMENT"]
		if field.IsPrimaryKey {
			autoIncrease = true
		}
		if v == "FALSE" {
			autoIncrease = false
		}

		sqlType = scope.Dialect().SqlTag(reflectValue, size, autoIncrease)
	}

	if strings.TrimSpace(additionalType) == "" {
		return sqlType
	} else {
		return fmt.Sprintf("%v %v", sqlType, additionalType)
	}
}

func parseTagSetting(tags reflect.StructTag) map[string]string {
	setting := map[string]string{}
	for _, str := range []string{tags.Get("sql"), tags.Get("gorm")} {
		tags := strings.Split(str, ";")
		for _, value := range tags {
			v := strings.Split(value, ":")
			k := strings.TrimSpace(strings.ToUpper(v[0]))
			if len(v) >= 2 {
				setting[k] = strings.Join(v[1:], ":")
			} else {
				setting[k] = k
			}
		}
	}
	return setting
}
