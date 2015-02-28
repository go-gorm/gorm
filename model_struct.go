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

type ModelStruct struct {
	PrimaryKeyField *StructField
	StructFields    []*StructField
	ModelType       reflect.Type
	TableName       string
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
	SqlTag          string
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
		SqlTag:          structField.SqlTag,
		Tag:             structField.Tag,
		Struct:          structField.Struct,
		IsForeignKey:    structField.IsForeignKey,
		Relationship:    structField.Relationship,
	}
}

type Relationship struct {
	Kind                        string
	PolymorphicType             string
	PolymorphicDBName           string
	ForeignFieldName            string
	ForeignDBName               string
	AssociationForeignFieldName string
	AssociationForeignDBName    string
	JoinTable                   string
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

	if scope.db != nil {
		if value, ok := scope.db.parent.ModelStructs[scopeType]; ok {
			return value
		}
	}

	modelStruct.ModelType = scopeType
	if scopeType.Kind() != reflect.Struct {
		return &modelStruct
	}

	// Set tablename
	if fm := reflect.New(scopeType).MethodByName("TableName"); fm.IsValid() {
		if results := fm.Call([]reflect.Value{}); len(results) > 0 {
			if name, ok := results[0].Interface().(string); ok {
				modelStruct.TableName = name
			}
		}
	} else {
		modelStruct.TableName = ToDBName(scopeType.Name())
		if scope.db == nil || !scope.db.parent.singularTable {
			for index, reg := range pluralMapKeys {
				if reg.MatchString(modelStruct.TableName) {
					modelStruct.TableName = reg.ReplaceAllString(modelStruct.TableName, pluralMapValues[index])
				}
			}
		}
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
					modelStruct.PrimaryKeyField = field
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

				foreignKey := gormSettings["FOREIGNKEY"]
				if polymorphic := gormSettings["POLYMORPHIC"]; polymorphic != "" {
					if polymorphicField := getForeignField(polymorphic+"Id", toScope.GetStructFields()); polymorphicField != nil {
						if polymorphicType := getForeignField(polymorphic+"Type", toScope.GetStructFields()); polymorphicType != nil {
							relationship.ForeignFieldName = polymorphicField.Name
							relationship.ForeignDBName = polymorphicField.DBName
							relationship.PolymorphicType = polymorphicType.Name
							relationship.PolymorphicDBName = polymorphicType.DBName
							polymorphicType.IsForeignKey = true
							polymorphicField.IsForeignKey = true
						}
					}
				}

				switch indirectType.Kind() {
				case reflect.Slice:
					elemType := indirectType.Elem()
					if elemType.Kind() == reflect.Ptr {
						elemType = elemType.Elem()
					}

					if elemType.Kind() == reflect.Struct {
						if foreignKey == "" {
							foreignKey = scopeType.Name() + "Id"
						}

						if many2many := gormSettings["MANY2MANY"]; many2many != "" {
							relationship.Kind = "many_to_many"
							relationship.JoinTable = many2many

							associationForeignKey := gormSettings["ASSOCIATIONFOREIGNKEY"]
							if associationForeignKey == "" {
								associationForeignKey = elemType.Name() + "Id"
							}

							relationship.ForeignFieldName = foreignKey
							relationship.ForeignDBName = ToDBName(foreignKey)
							relationship.AssociationForeignFieldName = associationForeignKey
							relationship.AssociationForeignDBName = ToDBName(associationForeignKey)
							field.Relationship = relationship
						} else {
							relationship.Kind = "has_many"
							if foreignField := getForeignField(foreignKey, toScope.GetStructFields()); foreignField != nil {
								relationship.ForeignFieldName = foreignField.Name
								relationship.ForeignDBName = foreignField.DBName
								foreignField.IsForeignKey = true
								field.Relationship = relationship
							} else if relationship.ForeignFieldName != "" {
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
								modelStruct.PrimaryKeyField = toField
							}
						}
						continue
					} else {
						belongsToForeignKey := foreignKey
						if belongsToForeignKey == "" {
							belongsToForeignKey = field.Name + "Id"
						}

						if foreignField := getForeignField(belongsToForeignKey, fields); foreignField != nil {
							relationship.Kind = "belongs_to"
							relationship.ForeignFieldName = foreignField.Name
							relationship.ForeignDBName = foreignField.DBName
							foreignField.IsForeignKey = true
							field.Relationship = relationship
						} else {
							if foreignKey == "" {
								foreignKey = modelStruct.ModelType.Name() + "Id"
							}
							relationship.Kind = "has_one"
							if foreignField := getForeignField(foreignKey, toScope.GetStructFields()); foreignField != nil {
								relationship.ForeignFieldName = foreignField.Name
								relationship.ForeignDBName = foreignField.DBName
								foreignField.IsForeignKey = true
								field.Relationship = relationship
							} else if relationship.ForeignFieldName != "" {
								field.Relationship = relationship
							}
						}
					}
				default:
					field.IsNormal = true
				}
			}

			if field.IsNormal {
				if modelStruct.PrimaryKeyField == nil && field.DBName == "id" {
					field.IsPrimaryKey = true
					modelStruct.PrimaryKeyField = field
				}

				if scope.db != nil {
					scope.generateSqlTag(field)
				}
			}
		}
		modelStruct.StructFields = append(modelStruct.StructFields, field)
	}

	if scope.db != nil {
		scope.db.parent.ModelStructs[scopeType] = &modelStruct
	}

	return &modelStruct
}

func (scope *Scope) GetStructFields() (fields []*StructField) {
	return scope.GetModelStruct().StructFields
}

func (scope *Scope) generateSqlTag(field *StructField) {
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
		additionalType = additionalType + "DEFAULT " + value
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

		if field.IsPrimaryKey {
			sqlType = scope.Dialect().PrimaryKeyTag(reflectValue, size)
		} else {
			sqlType = scope.Dialect().SqlTag(reflectValue, size)
		}
	}

	if strings.TrimSpace(additionalType) == "" {
		field.SqlTag = sqlType
	} else {
		field.SqlTag = fmt.Sprintf("%v %v", sqlType, additionalType)
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
