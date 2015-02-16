package gorm

import (
	"database/sql"
	"go/ast"
	"reflect"
	"strconv"
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
	SqlTag       string
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

func (scope *Scope) GetStructFields() (fields []*StructField) {
	reflectValue := reflect.Indirect(reflect.ValueOf(scope.Value))
	if reflectValue.Kind() == reflect.Slice {
		reflectValue = reflect.Indirect(reflect.New(reflectValue.Elem().Type()))
	}

	scopeTyp := reflectValue.Type()
	hasPrimaryKey := false
	for i := 0; i < scopeTyp.NumField(); i++ {
		fieldStruct := scopeTyp.Field(i)
		if !ast.IsExported(fieldStruct.Name) {
			continue
		}
		var field *StructField

		if fieldStruct.Tag.Get("sql") == "-" {
			field.IsIgnored = true
		} else {
			sqlSettings := parseTagSetting(fieldStruct.Tag.Get("sql"))
			settings := parseTagSetting(fieldStruct.Tag.Get("gorm"))
			if _, ok := settings["PRIMARY_KEY"]; ok {
				field.IsPrimaryKey = true
				hasPrimaryKey = true
			}

			if value, ok := sqlSettings["DEFAULT"]; ok {
				field.DefaultValue = &value
			}

			if value, ok := settings["COLUMN"]; ok {
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

			many2many := settings["MANY2MANY"]
			foreignKey := SnakeToUpperCamel(settings["FOREIGNKEY"])
			foreignType := SnakeToUpperCamel(settings["FOREIGNTYPE"])
			associationForeignKey := SnakeToUpperCamel(settings["ASSOCIATIONFOREIGNKEY"])
			if polymorphic := SnakeToUpperCamel(settings["POLYMORPHIC"]); polymorphic != "" {
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
						} else if !reflect.New(typ).FieldByName(foreignKey).IsValid() {
							foreignKey = ""
						}

						field.Relationship = &relationship{
							JoinTable:             many2many,
							ForeignKey:            foreignKey,
							ForeignType:           foreignType,
							AssociationForeignKey: associationForeignKey,
							Kind: kind,
						}
					} else {
						field.IsNormal = true
					}
				case reflect.Struct:
					if _, ok := settings["EMBEDDED"]; ok || fieldStruct.Anonymous {
						for _, field := range scope.New(reflect.New(indirectType).Interface()).GetStructFields() {
							fields = append(fields, field)
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

						field.Relationship = &relationship{ForeignKey: foreignKey, ForeignType: foreignType, Kind: kind}
					}

				default:
					field.IsNormal = true
				}
			}
		}
		fields = append(fields, field)
	}

	if !hasPrimaryKey {
		for _, field := range fields {
			if field.DBName == "id" {
				field.IsPrimaryKey = true
			}
		}
	}

	for _, field := range fields {
		var sqlType string
		size := 255
		sqlTag := field.Tag.Get("sql")
		sqlSetting = parseTagSetting(sqlTag)

		if value, ok := sqlSetting["SIZE"]; ok {
			if i, err := strconv.Atoi(value); err == nil {
				size = i
			} else {
				size = 0
			}
		}

		if value, ok := sqlSetting["TYPE"]; ok {
			typ = value
		}

		additionalType := sqlSetting["NOT NULL"] + " " + sqlSetting["UNIQUE"]
		if value, ok := sqlSetting["DEFAULT"]; ok {
			additionalType = additionalType + "DEFAULT " + value
		}

		if field.IsScanner {
			var getScannerValue func(reflect.Value)
			getScannerValue = func(reflectValue reflect.Value) {
				if _, isScanner := reflect.New(reflectValue.Type()).Interface().(sql.Scanner); isScanner {
					getScannerValue(reflectValue.Field(0))
				}
			}
			getScannerValue(reflectValue.Field(0))
		}
		if field.IsNormal {
			typ + " " + additionalType
		} else if !field.IsTime {
			return typ + " " + additionalType
		}

		if len(typ) == 0 {
			if field.IsPrimaryKey {
				typ = scope.Dialect().PrimaryKeyTag(reflectValue, size)
			} else {
				typ = scope.Dialect().SqlTag(reflectValue, size)
			}
		}

		return typ + " " + additionalType
	}
	return
}
