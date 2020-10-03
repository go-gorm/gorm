package schema

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"reflect"
	"sync"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
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
	namer                     Namer
	cacheStore                *sync.Map
}

func (schema Schema) String() string {
	if schema.ModelType.Name() == "" {
		return fmt.Sprintf("%v(%v)", schema.Name, schema.Table)
	}
	return fmt.Sprintf("%v.%v", schema.ModelType.PkgPath(), schema.ModelType.Name())
}

func (schema Schema) MakeSlice() reflect.Value {
	slice := reflect.MakeSlice(reflect.SliceOf(reflect.PtrTo(schema.ModelType)), 0, 0)
	results := reflect.New(slice.Type())
	results.Elem().Set(slice)
	return results
}

func (schema Schema) LookUpField(name string) *Field {
	if field, ok := schema.FieldsByDBName[name]; ok {
		return field
	}
	if field, ok := schema.FieldsByName[name]; ok {
		return field
	}
	return nil
}

type Tabler interface {
	TableName() string
}

// get data type from dialector
func Parse(dest interface{}, cacheStore *sync.Map, namer Namer) (*Schema, error) {
	if dest == nil {
		return nil, fmt.Errorf("%w: %+v", ErrUnsupportedDataType, dest)
	}

	modelType := reflect.ValueOf(dest).Type()
	for modelType.Kind() == reflect.Slice || modelType.Kind() == reflect.Array || modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	if modelType.Kind() != reflect.Struct {
		if modelType.PkgPath() == "" {
			return nil, fmt.Errorf("%w: %+v", ErrUnsupportedDataType, dest)
		}
		return nil, fmt.Errorf("%w: %v.%v", ErrUnsupportedDataType, modelType.PkgPath(), modelType.Name())
	}

	if v, ok := cacheStore.Load(modelType); ok {
		return v.(*Schema), nil
	}

	modelValue := reflect.New(modelType)
	tableName := namer.TableName(modelType.Name())
	if tabler, ok := modelValue.Interface().(Tabler); ok {
		tableName = tabler.TableName()
	}
	if en, ok := namer.(embeddedNamer); ok {
		tableName = en.Table
	}

	schema := &Schema{
		Name:           modelType.Name(),
		ModelType:      modelType,
		Table:          tableName,
		FieldsByName:   map[string]*Field{},
		FieldsByDBName: map[string]*Field{},
		Relationships:  Relationships{Relations: map[string]*Relationship{}},
		cacheStore:     cacheStore,
		namer:          namer,
	}

	defer func() {
		if schema.err != nil {
			logger.Default.Error(context.Background(), schema.err.Error())
			cacheStore.Delete(modelType)
		}
	}()

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

		if field.DBName != "" {
			// nonexistence or shortest path or first appear prioritized if has permission
			if v, ok := schema.FieldsByDBName[field.DBName]; !ok || ((field.Creatable || field.Updatable || field.Readable) && len(field.BindNames) < len(v.BindNames)) {
				if _, ok := schema.FieldsByDBName[field.DBName]; !ok {
					schema.DBNames = append(schema.DBNames, field.DBName)
				}
				schema.FieldsByDBName[field.DBName] = field
				schema.FieldsByName[field.Name] = field

				if v != nil && v.PrimaryKey {
					for idx, f := range schema.PrimaryFields {
						if f == v {
							schema.PrimaryFields = append(schema.PrimaryFields[0:idx], schema.PrimaryFields[idx+1:]...)
						}
					}
				}

				if field.PrimaryKey {
					schema.PrimaryFields = append(schema.PrimaryFields, field)
				}
			}
		}

		if _, ok := schema.FieldsByName[field.Name]; !ok {
			schema.FieldsByName[field.Name] = field
		}

		field.setupValuerAndSetter()
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

	if schema.PrioritizedPrimaryField == nil && len(schema.PrimaryFields) == 1 {
		schema.PrioritizedPrimaryField = schema.PrimaryFields[0]
	}

	for _, field := range schema.PrimaryFields {
		schema.PrimaryFieldDBNames = append(schema.PrimaryFieldDBNames, field.DBName)
	}

	for _, field := range schema.FieldsByDBName {
		if field.HasDefaultValue && field.DefaultValueInterface == nil {
			schema.FieldsWithDefaultDBValue = append(schema.FieldsWithDefaultDBValue, field)
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

	callbacks := []string{"BeforeCreate", "AfterCreate", "BeforeUpdate", "AfterUpdate", "BeforeSave", "AfterSave", "BeforeDelete", "AfterDelete", "AfterFind"}
	for _, name := range callbacks {
		if methodValue := modelValue.MethodByName(name); methodValue.IsValid() {
			switch methodValue.Type().String() {
			case "func(*gorm.DB) error": // TODO hack
				reflect.Indirect(reflect.ValueOf(schema)).FieldByName(name).SetBool(true)
			default:
				logger.Default.Warn(context.Background(), "Model %v don't match %vInterface, should be %v(*gorm.DB)", schema, name, name)
			}
		}
	}

	if _, loaded := cacheStore.LoadOrStore(modelType, schema); !loaded {
		if _, embedded := schema.cacheStore.Load(embeddedCacheKey); !embedded {
			for _, field := range schema.Fields {
				if field.DataType == "" && (field.Creatable || field.Updatable || field.Readable) {
					if schema.parseRelation(field); schema.err != nil {
						return schema, schema.err
					}
				}

				fieldValue := reflect.New(field.IndirectFieldType)
				if fc, ok := fieldValue.Interface().(CreateClausesInterface); ok {
					field.Schema.CreateClauses = append(field.Schema.CreateClauses, fc.CreateClauses(field)...)
				}

				if fc, ok := fieldValue.Interface().(QueryClausesInterface); ok {
					field.Schema.QueryClauses = append(field.Schema.QueryClauses, fc.QueryClauses(field)...)
				}

				if fc, ok := fieldValue.Interface().(UpdateClausesInterface); ok {
					field.Schema.UpdateClauses = append(field.Schema.UpdateClauses, fc.UpdateClauses(field)...)
				}

				if fc, ok := fieldValue.Interface().(DeleteClausesInterface); ok {
					field.Schema.DeleteClauses = append(field.Schema.DeleteClauses, fc.DeleteClauses(field)...)
				}
			}
		}
	}

	return schema, schema.err
}
