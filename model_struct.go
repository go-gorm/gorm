package gorm

import (
	"database/sql"
	"fmt"
	"go/ast"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var modelStructs = map[reflect.Type]*ModelStruct{}

var DefaultTableNameHandler = func(db *DB, defaultTableName string) string {
	return defaultTableName
}

type ModelStruct struct {
	PrimaryFields    []*StructField
	StructFields     []*StructField
	ModelType        reflect.Type
	defaultTableName string
}

func (s ModelStruct) TableName(db *DB) string {
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

var pluralMapKeys = []*regexp.Regexp{regexp.MustCompile("ch$"), regexp.MustCompile("ss$"), regexp.MustCompile("sh$"), regexp.MustCompile("day$"), regexp.MustCompile("y$"), regexp.MustCompile("x$"), regexp.MustCompile("([^s])s?$")}
var pluralMapValues = []string{"ches", "sses", "shes", "days", "ies", "xes", "${1}s"}

func (scope *Scope) GetModelStruct() *ModelStruct {
	var modelStruct ModelStruct

	reflectValue := reflect.Indirect(reflect.ValueOf(scope.Value))
	if !reflectValue.IsValid() {
		return &modelStruct
	}

	if reflectValue.Kind() == reflect.Slice {
		reflectValue = reflect.Indirect(reflect.New(reflectValue.Type().Elem()))
	}

	scopeType := reflectValue.Type()

	if scopeType.Kind() == reflect.Ptr {
		scopeType = scopeType.Elem()
	}

	if value, ok := modelStructs[scopeType]; ok {
		return value
	}

	modelStruct.ModelType = scopeType
	if scopeType.Kind() != reflect.Struct {
		return &modelStruct
	}

	// Set tablename
	type tabler interface {
		TableName() string
	}

	if tabler, ok := reflect.New(scopeType).Interface().(interface {
		TableName() string
	}); ok {
		modelStruct.defaultTableName = tabler.TableName()
	} else {
		name := ToDBName(scopeType.Name())
		if scope.db == nil || !scope.db.parent.singularTable {
			for index, reg := range pluralMapKeys {
				if reg.MatchString(name) {
					name = reg.ReplaceAllString(name, pluralMapValues[index])
				}
			}
		}

		modelStruct.defaultTableName = name
	}

	// Get all fields
	fields := []*StructField{}
	for i := 0; i < scopeType.NumField(); i++ {
		if fieldStruct := scopeType.Field(i); ast.IsExported(fieldStruct.Name) {
			field := &StructField{
				Struct: fieldStruct,
				Name:   fieldStruct.Name,
				Names:  []string{fieldStruct.Name},
				Tag:    fieldStruct.Tag,
			}

			if fieldStruct.Tag.Get("sql") == "-" {
				field.IsIgnored = true
			} else {
				sqlSettings := parseTagSetting(field.Tag.Get("sql"))
				gormSettings := parseTagSetting(field.Tag.Get("gorm"))
				if _, ok := gormSettings["PRIMARY_KEY"]; ok {
					field.IsPrimaryKey = true
					modelStruct.PrimaryFields = append(modelStruct.PrimaryFields, field)
				}

				if _, ok := sqlSettings["DEFAULT"]; ok {
					field.HasDefaultValue = true
				}

				if value, ok := gormSettings["COLUMN"]; ok {
					field.DBName = value
				} else {
					field.DBName = ToDBName(fieldStruct.Name)
				}
			}
			fields = append(fields, field)
		}
	}

	defer func() {
		for _, field := range fields {
			if !field.IsIgnored {
				fieldStruct := field.Struct
				fieldType, indirectType := fieldStruct.Type, fieldStruct.Type
				if indirectType.Kind() == reflect.Ptr {
					indirectType = indirectType.Elem()
				}

				if _, isScanner := reflect.New(fieldType).Interface().(sql.Scanner); isScanner {
					field.IsScanner, field.IsNormal = true, true
				}

				if _, isTime := reflect.New(indirectType).Interface().(*time.Time); isTime {
					field.IsNormal = true
				}

				if !field.IsNormal {
					gormSettings := parseTagSetting(field.Tag.Get("gorm"))
					toScope := scope.New(reflect.New(fieldStruct.Type).Interface())

					getForeignField := func(column string, fields []*StructField) *StructField {
						for _, field := range fields {
							if field.Name == column || field.DBName == ToDBName(column) {
								return field
							}
						}
						return nil
					}

					var relationship = &Relationship{}

					if polymorphic := gormSettings["POLYMORPHIC"]; polymorphic != "" {
						if polymorphicField := getForeignField(polymorphic+"Id", toScope.GetStructFields()); polymorphicField != nil {
							if polymorphicType := getForeignField(polymorphic+"Type", toScope.GetStructFields()); polymorphicType != nil {
								relationship.ForeignFieldNames = []string{polymorphicField.Name}
								relationship.ForeignDBNames = []string{polymorphicField.DBName}
								relationship.AssociationForeignFieldNames = []string{scope.PrimaryField().Name}
								relationship.AssociationForeignDBNames = []string{scope.PrimaryField().DBName}
								relationship.PolymorphicType = polymorphicType.Name
								relationship.PolymorphicDBName = polymorphicType.DBName
								polymorphicType.IsForeignKey = true
								polymorphicField.IsForeignKey = true
							}
						}
					}

					var foreignKeys []string
					if foreignKey, ok := gormSettings["FOREIGNKEY"]; ok {
						foreignKeys = append(foreignKeys, foreignKey)
					}
					switch indirectType.Kind() {
					case reflect.Slice:
						elemType := indirectType.Elem()
						if elemType.Kind() == reflect.Ptr {
							elemType = elemType.Elem()
						}

						if elemType.Kind() == reflect.Struct {
							if many2many := gormSettings["MANY2MANY"]; many2many != "" {
								relationship.Kind = "many_to_many"

								// foreign keys
								if len(foreignKeys) == 0 {
									for _, field := range scope.PrimaryFields() {
										foreignKeys = append(foreignKeys, field.DBName)
									}
								}

								for _, foreignKey := range foreignKeys {
									if field, ok := scope.FieldByName(foreignKey); ok {
										relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, field.DBName)
										joinTableDBName := ToDBName(scopeType.Name()) + "_" + field.DBName
										relationship.ForeignDBNames = append(relationship.ForeignDBNames, joinTableDBName)
									}
								}

								// association foreign keys
								var associationForeignKeys []string
								if foreignKey := gormSettings["ASSOCIATIONFOREIGNKEY"]; foreignKey != "" {
									associationForeignKeys = []string{gormSettings["ASSOCIATIONFOREIGNKEY"]}
								} else {
									for _, field := range toScope.PrimaryFields() {
										associationForeignKeys = append(associationForeignKeys, field.DBName)
									}
								}

								for _, name := range associationForeignKeys {
									if field, ok := toScope.FieldByName(name); ok {
										relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, field.DBName)
										joinTableDBName := ToDBName(elemType.Name()) + "_" + field.DBName
										relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, joinTableDBName)
									}
								}

								joinTableHandler := JoinTableHandler{}
								joinTableHandler.Setup(relationship, many2many, scopeType, elemType)
								relationship.JoinTableHandler = &joinTableHandler
								field.Relationship = relationship
							} else {
								relationship.Kind = "has_many"

								if len(foreignKeys) == 0 {
									for _, field := range scope.PrimaryFields() {
										if foreignField := getForeignField(scopeType.Name()+field.Name, toScope.GetStructFields()); foreignField != nil {
											relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, field.Name)
											relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, field.DBName)
											relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, foreignField.Name)
											relationship.ForeignDBNames = append(relationship.ForeignDBNames, foreignField.DBName)
											foreignField.IsForeignKey = true
										}
									}
								} else {
									for _, foreignKey := range foreignKeys {
										if foreignField := getForeignField(foreignKey, toScope.GetStructFields()); foreignField != nil {
											relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, scope.PrimaryField().Name)
											relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, scope.PrimaryField().DBName)
											relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, foreignField.Name)
											relationship.ForeignDBNames = append(relationship.ForeignDBNames, foreignField.DBName)
											foreignField.IsForeignKey = true
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
					case reflect.Struct:
						if _, ok := gormSettings["EMBEDDED"]; ok || fieldStruct.Anonymous {
							for _, toField := range toScope.GetStructFields() {
								toField = toField.clone()
								toField.Names = append([]string{fieldStruct.Name}, toField.Names...)
								modelStruct.StructFields = append(modelStruct.StructFields, toField)
								if toField.IsPrimaryKey {
									modelStruct.PrimaryFields = append(modelStruct.PrimaryFields, toField)
								}
							}
							continue
						} else {
							if len(foreignKeys) == 0 {
								for _, f := range scope.PrimaryFields() {
									if foreignField := getForeignField(modelStruct.ModelType.Name()+f.Name, toScope.GetStructFields()); foreignField != nil {
										relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, f.Name)
										relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, f.DBName)
										relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, foreignField.Name)
										relationship.ForeignDBNames = append(relationship.ForeignDBNames, foreignField.DBName)
										foreignField.IsForeignKey = true
									}
								}
							} else {
								for _, foreignKey := range foreignKeys {
									if foreignField := getForeignField(foreignKey, toScope.GetStructFields()); foreignField != nil {
										relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, scope.PrimaryField().Name)
										relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, scope.PrimaryField().DBName)
										relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, foreignField.Name)
										relationship.ForeignDBNames = append(relationship.ForeignDBNames, foreignField.DBName)
										foreignField.IsForeignKey = true
									}
								}
							}

							if len(relationship.ForeignFieldNames) != 0 {
								relationship.Kind = "has_one"
								field.Relationship = relationship
							} else {
								if len(foreignKeys) == 0 {
									for _, f := range toScope.PrimaryFields() {
										if foreignField := getForeignField(field.Name+f.Name, fields); foreignField != nil {
											relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, f.Name)
											relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, f.DBName)
											relationship.ForeignFieldNames = append(relationship.ForeignFieldNames, foreignField.Name)
											relationship.ForeignDBNames = append(relationship.ForeignDBNames, foreignField.DBName)
											foreignField.IsForeignKey = true
										}
									}
								} else {
									for _, foreignKey := range foreignKeys {
										if foreignField := getForeignField(foreignKey, fields); foreignField != nil {
											relationship.AssociationForeignFieldNames = append(relationship.AssociationForeignFieldNames, toScope.PrimaryField().Name)
											relationship.AssociationForeignDBNames = append(relationship.AssociationForeignDBNames, toScope.PrimaryField().DBName)
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
						}
					default:
						field.IsNormal = true
					}
				}

				if field.IsNormal {
					if len(modelStruct.PrimaryFields) == 0 && field.DBName == "id" {
						field.IsPrimaryKey = true
						modelStruct.PrimaryFields = append(modelStruct.PrimaryFields, field)
					}
				}
			}
			modelStruct.StructFields = append(modelStruct.StructFields, field)
		}
	}()

	modelStructs[scopeType] = &modelStruct

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
	sqlSettings := parseTagSetting(field.Tag.Get("sql"))

	if value, ok := sqlSettings["TYPE"]; ok {
		sqlType = value
	}

	additionalType := sqlSettings["NOT NULL"] + " " + sqlSettings["UNIQUE"]
	if value, ok := sqlSettings["DEFAULT"]; ok {
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

		if value, ok := sqlSettings["SIZE"]; ok {
			size, _ = strconv.Atoi(value)
		}

		_, autoIncrease := sqlSettings["AUTO_INCREMENT"]
		if field.IsPrimaryKey {
			autoIncrease = true
		}

		sqlType = scope.Dialect().SqlTag(reflectValue, size, autoIncrease)
	}

	if strings.TrimSpace(additionalType) == "" {
		return sqlType
	} else {
		return fmt.Sprintf("%v %v", sqlType, additionalType)
	}
}

func parseTagSetting(str string) map[string]string {
	tags := strings.Split(str, ";")
	setting := map[string]string{}
	for _, value := range tags {
		v := strings.Split(value, ":")
		k := strings.TrimSpace(strings.ToUpper(v[0]))
		if len(v) == 2 {
			setting[k] = v[1]
		} else {
			setting[k] = k
		}
	}
	return setting
}
