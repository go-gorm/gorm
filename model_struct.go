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
	TableName       string
}

type StructField struct {
	Name         string
	DBName       string
	IsPrimaryKey bool
	IsScanner    bool
	IsTime       bool
	IsNormal     bool
	IsIgnored    bool
	DefaultValue *string
	GormSettings map[string]string
	SqlSettings  map[string]string
	SqlTag       string
	Struct       reflect.StructField
	Relationship *Relationship
}

type Relationship struct {
	Kind                        string
	ForeignType                 string
	ForeignFieldName            string
	ForeignDBName               string
	AssociationForeignFieldName string
	AssociationForeignDBName    string
	JoinTable                   string
}

func (scope *Scope) GenerateSqlTag(field *StructField) {
	var sqlType string
	reflectValue := reflect.Indirect(reflect.New(field.Struct.Type))

	if value, ok := field.SqlSettings["TYPE"]; ok {
		sqlType = value
	}

	additionalType := field.SqlSettings["NOT NULL"] + " " + field.SqlSettings["UNIQUE"]
	if value, ok := field.SqlSettings["DEFAULT"]; ok {
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

		if value, ok := field.SqlSettings["SIZE"]; ok {
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

var pluralMapKeys = []*regexp.Regexp{regexp.MustCompile("ch$"), regexp.MustCompile("ss$"), regexp.MustCompile("sh$"), regexp.MustCompile("day$"), regexp.MustCompile("y$"), regexp.MustCompile("x$"), regexp.MustCompile("([^s])s?$")}
var pluralMapValues = []string{"ches", "sses", "shes", "days", "ies", "xes", "${1}s"}

func (scope *Scope) GetModelStruct() *ModelStruct {
	var modelStruct ModelStruct

	reflectValue := reflect.Indirect(reflect.ValueOf(scope.Value))
	if reflectValue.Kind() == reflect.Slice {
		reflectValue = reflect.Indirect(reflect.New(reflectValue.Elem().Type()))
	}
	scopeTyp := reflectValue.Type()

	// Set tablename
	if fm := reflect.New(scopeTyp).MethodByName("TableName"); fm.IsValid() {
		if results := fm.Call([]reflect.Value{}); len(results) > 0 {
			if name, ok := results[0].Interface().(string); ok {
				modelStruct.TableName = name
			}
		}
	} else {
		modelStruct.TableName = ToSnake(scopeTyp.Name())
		if scope.db == nil || !scope.db.parent.singularTable {
			for index, reg := range pluralMapKeys {
				if reg.MatchString(modelStruct.TableName) {
					modelStruct.TableName = reg.ReplaceAllString(modelStruct.TableName, pluralMapValues[index])
				}
			}
		}
	}

	// Set fields
	for i := 0; i < scopeTyp.NumField(); i++ {
		fieldStruct := scopeTyp.Field(i)
		if !ast.IsExported(fieldStruct.Name) {
			continue
		}

		field := &StructField{Struct: fieldStruct}
		if fieldStruct.Tag.Get("sql") == "-" {
			field.IsIgnored = true
		} else {
			field.SqlSettings = parseTagSetting(fieldStruct.Tag.Get("sql"))
			field.GormSettings = parseTagSetting(fieldStruct.Tag.Get("gorm"))
			if _, ok := field.GormSettings["PRIMARY_KEY"]; ok {
				field.IsPrimaryKey = true
				modelStruct.PrimaryKeyField = field
			}

			if value, ok := field.SqlSettings["DEFAULT"]; ok {
				field.DefaultValue = &value
			}

			if value, ok := field.GormSettings["COLUMN"]; ok {
				field.DBName = value
			} else {
				field.DBName = ToSnake(fieldStruct.Name)
			}

			fieldType, indirectType := fieldStruct.Type, fieldStruct.Type
			if indirectType.Kind() == reflect.Ptr {
				indirectType = indirectType.Elem()
			}

			if _, isScanner := reflect.New(fieldType).Interface().(sql.Scanner); isScanner {
				field.IsScanner, field.IsNormal = true, true
			}

			if _, isTime := reflect.New(indirectType).Interface().(time.Time); isTime {
				field.IsTime, field.IsNormal = true, true
			}

			many2many := field.GormSettings["MANY2MANY"]
			foreignKey := SnakeToUpperCamel(field.GormSettings["FOREIGNKEY"])
			foreignType := SnakeToUpperCamel(field.GormSettings["FOREIGNTYPE"])
			associationForeignKey := SnakeToUpperCamel(field.GormSettings["ASSOCIATIONFOREIGNKEY"])
			if polymorphic := SnakeToUpperCamel(field.GormSettings["POLYMORPHIC"]); polymorphic != "" {
				foreignKey = polymorphic + "Id"
				foreignType = polymorphic + "Type"
			}

			if !field.IsNormal {
				switch indirectType.Kind() {
				case reflect.Slice:
					typ := indirectType.Elem()
					if typ.Kind() == reflect.Ptr {
						typ = typ.Elem()
					}

					if typ.Kind() == reflect.Struct {
						kind := "has_many"

						if foreignKey == "" {
							foreignKey = indirectType.Name() + "Id"
						}

						if associationForeignKey == "" {
							associationForeignKey = typ.Name() + "Id"
						}

						if many2many != "" {
							kind = "many_to_many"
						} else if !reflect.New(typ).Elem().FieldByName(foreignKey).IsValid() {
							foreignKey = ""
						}

						field.Relationship = &Relationship{
							JoinTable:                   many2many,
							ForeignType:                 foreignType,
							ForeignFieldName:            foreignKey,
							AssociationForeignFieldName: associationForeignKey,
							Kind: kind,
						}
					} else {
						field.IsNormal = true
					}
				case reflect.Struct:
					if _, ok := field.GormSettings["EMBEDDED"]; ok || fieldStruct.Anonymous {
						for _, field := range scope.New(reflect.New(indirectType).Interface()).GetStructFields() {
							modelStruct.StructFields = append(modelStruct.StructFields, field)
						}
						break
					} else {
						var belongsToForeignKey, hasOneForeignKey, kind string

						if foreignKey == "" {
							belongsToForeignKey = indirectType.Name() + "Id"
							hasOneForeignKey = scopeTyp.Name() + "Id"
						} else {
							belongsToForeignKey = foreignKey
							hasOneForeignKey = foreignKey
						}

						if _, ok := scopeTyp.FieldByName(belongsToForeignKey); ok {
							foreignKey = belongsToForeignKey
							kind = "belongs_to"
						} else {
							foreignKey = hasOneForeignKey
							kind = "has_one"
						}

						field.Relationship = &Relationship{ForeignFieldName: foreignKey, ForeignType: foreignType, Kind: kind}
					}

				default:
					field.IsNormal = true
				}
			}
		}
		modelStruct.StructFields = append(modelStruct.StructFields, field)
	}

	for _, field := range modelStruct.StructFields {
		if field.IsNormal {
			if modelStruct.PrimaryKeyField == nil && field.DBName == "id" {
				field.IsPrimaryKey = true
				modelStruct.PrimaryKeyField = field
			}

			scope.GenerateSqlTag(field)
		}
	}

	return &modelStruct
}

func (scope *Scope) GetStructFields() (fields []*StructField) {
	return scope.GetModelStruct().StructFields
}
