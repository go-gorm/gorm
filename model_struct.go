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
	cached           bool
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
	fields := []*StructField{}
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
			}

			if _, ok := field.TagSettings["PRIMARY_KEY"]; ok {
				field.IsPrimaryKey = true
				modelStruct.PrimaryFields = append(modelStruct.PrimaryFields, field)
			}

			if _, ok := field.TagSettings["DEFAULT"]; ok {
				field.HasDefaultValue = true
			}

			if value, ok := field.TagSettings["COLUMN"]; ok {
				field.DBName = value
			} else {
				field.DBName = ToDBName(fieldStruct.Name)
			}

			fields = append(fields, field)
		}
	}

	var finished = make(chan bool)
	go func(finished chan bool) {
		for _, field := range fields {
			if !field.IsIgnored {
				fieldStruct := field.Struct
				indirectType := fieldStruct.Type
				if indirectType.Kind() == reflect.Ptr {
					indirectType = indirectType.Elem()
				}

				if _, isScanner := reflect.New(indirectType).Interface().(sql.Scanner); isScanner {
					field.IsScanner, field.IsNormal = true, true
				}

				if _, isTime := reflect.New(indirectType).Interface().(*time.Time); isTime {
					field.IsNormal = true
				}

				if !field.IsNormal {
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

					if polymorphic := field.TagSettings["POLYMORPHIC"]; polymorphic != "" {
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
					if foreignKey, ok := field.TagSettings["FOREIGNKEY"]; ok {
						foreignKeys = append(foreignKeys, foreignKey)
					}
					switch indirectType.Kind() {
					case reflect.Slice:
						elemType := indirectType.Elem()
						if elemType.Kind() == reflect.Ptr {
							elemType = elemType.Elem()
						}

						if elemType.Kind() == reflect.Struct {
							if many2many := field.TagSettings["MANY2MANY"]; many2many != "" {
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
										joinTableDBName := ToDBName(reflectType.Name()) + "_" + field.DBName
										relationship.ForeignDBNames = append(relationship.ForeignDBNames, joinTableDBName)
									}
								}

								// association foreign keys
								var associationForeignKeys []string
								if foreignKey := field.TagSettings["ASSOCIATIONFOREIGNKEY"]; foreignKey != "" {
									associationForeignKeys = []string{foreignKey}
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
								joinTableHandler.Setup(relationship, many2many, reflectType, elemType)
								relationship.JoinTableHandler = &joinTableHandler
								field.Relationship = relationship
							} else {
								relationship.Kind = "has_many"

								if len(foreignKeys) == 0 {
									for _, field := range scope.PrimaryFields() {
										if foreignField := getForeignField(reflectType.Name()+field.Name, toScope.GetStructFields()); foreignField != nil {
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
						if _, ok := field.TagSettings["EMBEDDED"]; ok || fieldStruct.Anonymous {
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
		finished <- true
	}(finished)

	modelStructsMap.Set(reflectType, &modelStruct)

	<-finished
	modelStruct.cached = true

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
