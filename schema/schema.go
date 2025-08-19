package schema

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"path"
	"reflect"
	"strings"
	"sync"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type callbackType string

const (
	callbackTypeBeforeCreate callbackType = "BeforeCreate"
	callbackTypeBeforeUpdate callbackType = "BeforeUpdate"
	callbackTypeAfterCreate  callbackType = "AfterCreate"
	callbackTypeAfterUpdate  callbackType = "AfterUpdate"
	callbackTypeBeforeSave   callbackType = "BeforeSave"
	callbackTypeAfterSave    callbackType = "AfterSave"
	callbackTypeBeforeDelete callbackType = "BeforeDelete"
	callbackTypeAfterDelete  callbackType = "AfterDelete"
	callbackTypeAfterFind    callbackType = "AfterFind"
)

// ErrUnsupportedDataType unsupported data type
var ErrUnsupportedDataType = errors.New("unsupported data type")

type Schema struct {
	Name                      string
	ModelType                 reflect.Type
	Table                     string
	PrioritizedPrimaryField   *Field
	DBNames                   []string
	PrimaryFields             []*Field
	PrimaryFieldDBNames       []string
	Fields                    []*Field
	FieldsByName              map[string]*Field
	FieldsByBindName          map[string]*Field // embedded fields is 'Embed.Field'
	FieldsByDBName            map[string]*Field
	FieldsWithDefaultDBValue  []*Field // fields with default value assigned by database
	Relationships             Relationships
	CreateClauses             []clause.Interface
	QueryClauses              []clause.Interface
	UpdateClauses             []clause.Interface
	DeleteClauses             []clause.Interface
	BeforeCreate, AfterCreate bool
	BeforeUpdate, AfterUpdate bool
	BeforeDelete, AfterDelete bool
	BeforeSave, AfterSave     bool
	AfterFind                 bool
	err                       error
	initialized               chan struct{}
	namer                     Namer
	cacheStore                *sync.Map
}

func (schema *Schema) String() string {
	if schema.ModelType.Name() == "" {
		return fmt.Sprintf("%s(%s)", schema.Name, schema.Table)
	}
	return fmt.Sprintf("%s.%s", schema.ModelType.PkgPath(), schema.ModelType.Name())
}

func (schema *Schema) MakeSlice() reflect.Value {
	slice := reflect.MakeSlice(reflect.SliceOf(reflect.PointerTo(schema.ModelType)), 0, 20)
	results := reflect.New(slice.Type())
	results.Elem().Set(slice)

	return results
}

func (schema *Schema) LookUpField(name string) *Field {
	if field, ok := schema.FieldsByDBName[name]; ok {
		return field
	}
	if field, ok := schema.FieldsByName[name]; ok {
		return field
	}
	return nil
}

// LookUpFieldByBindName looks for the closest field in the embedded struct.
//
//	type Struct struct {
//		Embedded struct {
//			ID string // is selected by LookUpFieldByBindName([]string{"Embedded", "ID"}, "ID")
//		}
//		ID string // is selected by LookUpFieldByBindName([]string{"ID"}, "ID")
//	}
func (schema *Schema) LookUpFieldByBindName(bindNames []string, name string) *Field {
	for i := len(bindNames) - 1; i >= 0; i-- {
		find := strings.Join(bindNames[:i], ".") + "." + name
		if field, ok := schema.FieldsByBindName[find]; ok {
			return field
		}
	}
	return nil
}

type Tabler interface {
	TableName() string
}

type TablerWithNamer interface {
	TableName(Namer) string
}

var callbackTypes = []callbackType{
	callbackTypeBeforeCreate, callbackTypeAfterCreate,
	callbackTypeBeforeUpdate, callbackTypeAfterUpdate,
	callbackTypeBeforeSave, callbackTypeAfterSave,
	callbackTypeBeforeDelete, callbackTypeAfterDelete,
	callbackTypeAfterFind,
}

// Parse get data type from dialector
func Parse(dest interface{}, cacheStore *sync.Map, namer Namer) (*Schema, error) {
	return ParseWithSpecialTableName(dest, cacheStore, namer, "")
}

// ParseWithSpecialTableName get data type from dialector with extra schema table
func ParseWithSpecialTableName(dest interface{}, cacheStore *sync.Map, namer Namer, specialTableName string) (*Schema, error) {
	if dest == nil {
		return nil, fmt.Errorf("%w: %+v", ErrUnsupportedDataType, dest)
	}

	modelType := reflect.ValueOf(dest).Type()
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	if modelType.Kind() != reflect.Struct {
		if modelType.Kind() == reflect.Interface {
			modelType = reflect.Indirect(reflect.ValueOf(dest)).Elem().Type()
		}

		for modelType.Kind() == reflect.Slice || modelType.Kind() == reflect.Array || modelType.Kind() == reflect.Ptr {
			modelType = modelType.Elem()
		}

		if modelType.Kind() != reflect.Struct {
			if modelType.PkgPath() == "" {
				return nil, fmt.Errorf("%w: %+v", ErrUnsupportedDataType, dest)
			}
			return nil, fmt.Errorf("%w: %s.%s", ErrUnsupportedDataType, modelType.PkgPath(), modelType.Name())
		}
	}

	// Cache the Schema for performance,
	// Use the modelType or modelType + schemaTable (if it present) as cache key.
	var schemaCacheKey interface{} = modelType
	if specialTableName != "" {
		schemaCacheKey = fmt.Sprintf("%p-%s", modelType, specialTableName)
	}

	// Load exist schema cache, return if exists
	if v, ok := cacheStore.Load(schemaCacheKey); ok {
		s := v.(*Schema)
		// Wait for the initialization of other goroutines to complete
		<-s.initialized
		return s, s.err
	}

	var tableName string
	modelValue := reflect.New(modelType)
	if specialTableName != "" {
		tableName = specialTableName
	} else if en, ok := namer.(embeddedNamer); ok {
		tableName = en.Table
	} else if tabler, ok := modelValue.Interface().(Tabler); ok {
		tableName = tabler.TableName()
	} else if tabler, ok := modelValue.Interface().(TablerWithNamer); ok {
		tableName = tabler.TableName(namer)
	} else {
		tableName = namer.TableName(modelType.Name())
	}

	schema := &Schema{
		Name:             modelType.Name(),
		ModelType:        modelType,
		Table:            tableName,
		DBNames:          make([]string, 0, 10),
		Fields:           make([]*Field, 0, 10),
		FieldsByName:     make(map[string]*Field, 10),
		FieldsByBindName: make(map[string]*Field, 10),
		FieldsByDBName:   make(map[string]*Field, 10),
		Relationships:    Relationships{Relations: map[string]*Relationship{}},
		cacheStore:       cacheStore,
		namer:            namer,
		initialized:      make(chan struct{}),
	}
	// When the schema initialization is completed, the channel will be closed
	defer close(schema.initialized)

	// Load exist schema cache, return if exists
	if v, ok := cacheStore.Load(schemaCacheKey); ok {
		s := v.(*Schema)
		// Wait for the initialization of other goroutines to complete
		<-s.initialized
		return s, s.err
	}

	for i := 0; i < modelType.NumField(); i++ {
		if fieldStruct := modelType.Field(i); ast.IsExported(fieldStruct.Name) {
			if field := schema.ParseField(fieldStruct); field.EmbeddedSchema != nil {
				schema.Fields = append(schema.Fields, field.EmbeddedSchema.Fields...)
			} else {
				schema.Fields = append(schema.Fields, field)
			}
		}
	}

	for _, field := range schema.Fields {
		if field.DBName == "" && field.DataType != "" {
			field.DBName = namer.ColumnName(schema.Table, field.Name)
		}

		bindName := field.BindName()
		if field.DBName != "" {
			// nonexistence or shortest path or first appear prioritized if has permission
			if v, ok := schema.FieldsByDBName[field.DBName]; !ok || ((field.Creatable || field.Updatable || field.Readable) && len(field.BindNames) < len(v.BindNames)) {
				if _, ok := schema.FieldsByDBName[field.DBName]; !ok {
					schema.DBNames = append(schema.DBNames, field.DBName)
				}
				schema.FieldsByDBName[field.DBName] = field
				schema.FieldsByName[field.Name] = field
				schema.FieldsByBindName[bindName] = field

				if v != nil && v.PrimaryKey {
					// remove the existing primary key field
					for idx, f := range schema.PrimaryFields {
						if f.DBName == v.DBName {
							schema.PrimaryFields = append(schema.PrimaryFields[0:idx], schema.PrimaryFields[idx+1:]...)
						}
					}
				}

				if field.PrimaryKey {
					schema.PrimaryFields = append(schema.PrimaryFields, field)
				}
			}
		}

		if of, ok := schema.FieldsByName[field.Name]; !ok || of.TagSettings["-"] == "-" {
			schema.FieldsByName[field.Name] = field
		}
		if of, ok := schema.FieldsByBindName[bindName]; !ok || of.TagSettings["-"] == "-" {
			schema.FieldsByBindName[bindName] = field
		}

		field.setupValuerAndSetter(modelType)
	}

	prioritizedPrimaryField := schema.LookUpField("id")
	if prioritizedPrimaryField == nil {
		prioritizedPrimaryField = schema.LookUpField("ID")
	}

	if prioritizedPrimaryField != nil {
		if prioritizedPrimaryField.PrimaryKey {
			schema.PrioritizedPrimaryField = prioritizedPrimaryField
		} else if len(schema.PrimaryFields) == 0 {
			prioritizedPrimaryField.PrimaryKey = true
			schema.PrioritizedPrimaryField = prioritizedPrimaryField
			schema.PrimaryFields = append(schema.PrimaryFields, prioritizedPrimaryField)
		}
	}

	if schema.PrioritizedPrimaryField == nil {
		if len(schema.PrimaryFields) == 1 {
			schema.PrioritizedPrimaryField = schema.PrimaryFields[0]
		} else if len(schema.PrimaryFields) > 1 {
			// If there are multiple primary keys, the AUTOINCREMENT field is prioritized
			for _, field := range schema.PrimaryFields {
				if field.AutoIncrement {
					schema.PrioritizedPrimaryField = field
					break
				}
			}
		}
	}

	for _, field := range schema.PrimaryFields {
		schema.PrimaryFieldDBNames = append(schema.PrimaryFieldDBNames, field.DBName)
	}

	_, embedded := schema.cacheStore.Load(embeddedCacheKey)
	relationshipFields := []*Field{}
	for _, field := range schema.Fields {
		if field.DataType != "" && field.HasDefaultValue && field.DefaultValueInterface == nil {
			schema.FieldsWithDefaultDBValue = append(schema.FieldsWithDefaultDBValue, field)
		}

		if !embedded {
			if field.DataType == "" && field.GORMDataType == "" && (field.Creatable || field.Updatable || field.Readable) {
				relationshipFields = append(relationshipFields, field)
				schema.FieldsByName[field.Name] = field
				schema.FieldsByBindName[field.BindName()] = field
			}

			fieldValue := reflect.New(field.IndirectFieldType).Interface()
			if fc, ok := fieldValue.(CreateClausesInterface); ok {
				field.Schema.CreateClauses = append(field.Schema.CreateClauses, fc.CreateClauses(field)...)
			}

			if fc, ok := fieldValue.(QueryClausesInterface); ok {
				field.Schema.QueryClauses = append(field.Schema.QueryClauses, fc.QueryClauses(field)...)
			}

			if fc, ok := fieldValue.(UpdateClausesInterface); ok {
				field.Schema.UpdateClauses = append(field.Schema.UpdateClauses, fc.UpdateClauses(field)...)
			}

			if fc, ok := fieldValue.(DeleteClausesInterface); ok {
				field.Schema.DeleteClauses = append(field.Schema.DeleteClauses, fc.DeleteClauses(field)...)
			}
		}
	}

	if field := schema.PrioritizedPrimaryField; field != nil {
		switch field.GORMDataType {
		case Int, Uint:
			if _, ok := field.TagSettings["AUTOINCREMENT"]; !ok {
				if !field.HasDefaultValue || field.DefaultValueInterface != nil {
					schema.FieldsWithDefaultDBValue = append(schema.FieldsWithDefaultDBValue, field)
				}

				field.HasDefaultValue = true
				field.AutoIncrement = true
			}
		}
	}

	// Cache the schema
	if v, loaded := cacheStore.LoadOrStore(schemaCacheKey, schema); loaded {
		s := v.(*Schema)
		// Wait for the initialization of other goroutines to complete
		<-s.initialized
		return s, s.err
	}

	defer func() {
		if schema.err != nil {
			logger.Default.Error(context.Background(), schema.err.Error())
			cacheStore.Delete(modelType)
		}
	}()

	for _, cbName := range callbackTypes {
		if methodValue := modelValue.MethodByName(string(cbName)); methodValue.IsValid() {
			switch methodValue.Type().String() {
			case "func(*gorm.DB) error":
				expectedPkgPath := path.Dir(reflect.TypeOf(schema).Elem().PkgPath())
				if inVarPkg := methodValue.Type().In(0).Elem().PkgPath(); inVarPkg == expectedPkgPath {
					reflect.Indirect(reflect.ValueOf(schema)).FieldByName(string(cbName)).SetBool(true)
				} else {
					logger.Default.Warn(context.Background(), "In model %v, the hook function `%v(*gorm.DB) error` has an incorrect parameter type. The expected parameter type is `%v`, but the provided type is `%v`.", schema, cbName, expectedPkgPath, inVarPkg)
					// PASS
				}
			default:
				logger.Default.Warn(context.Background(), "Model %v don't match %vInterface, should be `%v(*gorm.DB) error`. Please see https://gorm.io/docs/hooks.html", schema, cbName, cbName)
			}
		}
	}

	// parse relationships
	for _, field := range relationshipFields {
		if schema.parseRelation(field); schema.err != nil {
			return schema, schema.err
		}
	}

	return schema, schema.err
}

func getOrParse(dest interface{}, cacheStore *sync.Map, namer Namer) (*Schema, error) {
	modelType := reflect.ValueOf(dest).Type()

	if modelType.Kind() != reflect.Struct {
		for modelType.Kind() == reflect.Slice || modelType.Kind() == reflect.Array || modelType.Kind() == reflect.Ptr {
			modelType = modelType.Elem()
		}

		if modelType.Kind() != reflect.Struct {
			if modelType.PkgPath() == "" {
				return nil, fmt.Errorf("%w: %+v", ErrUnsupportedDataType, dest)
			}
			return nil, fmt.Errorf("%w: %s.%s", ErrUnsupportedDataType, modelType.PkgPath(), modelType.Name())
		}
	}

	if v, ok := cacheStore.Load(modelType); ok {
		return v.(*Schema), nil
	}

	return Parse(dest, cacheStore, namer)
}
