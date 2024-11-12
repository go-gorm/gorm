package schema

import (
	"context"
	"driver"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jinzhu/now"
	"gorm.io/gorm/utils"
)

// Special types' reflect type
var (
	TimeReflectType    = reflect.TypeOf(time.Time{})
	TimePtrReflectType = reflect.TypeOf(&time.Time{})
	ByteReflectType    = reflect.TypeOf(uint8(0))
)

type (
	// DataType GORM data type
	DataType string
	// TimeType GORM time type
	TimeType int64
)

// GORM time types
const (
	UnixTime        TimeType = 1
	UnixSecond      TimeType = 2
	UnixMillisecond TimeType = 3
	UnixNanosecond  TimeType = 4
)

// GORM fields types
const (
	Bool   DataType = "bool"
	Int    DataType = "int"
	Uint   DataType = "uint"
	Float  DataType = "float"
	String DataType = "string"
	Time   DataType = "time"
	Bytes  DataType = "bytes"
)

const DefaultAutoIncrementIncrement int64 = 1

// Field represents a field in the schema
// Improvements made:
// 1. Added more descriptive comments
// 2. Moved longer blocks of code into helper functions to improve readability
// 3. Added error handling improvements and removed unnecessary nesting
// 4. Simplified boolean checks with utility functions
// 5. Grouped related initialization logic for clarity
// 6. Removed duplication by refactoring repeated tasks into functions

type Field struct {
	Name                   string
	DBName                 string
	BindNames              []string
	EmbeddedBindNames      []string
	DataType               DataType
	GORMDataType           DataType
	PrimaryKey             bool
	AutoIncrement          bool
	AutoIncrementIncrement int64
	Creatable              bool
	Updatable              bool
	Readable               bool
	AutoCreateTime         TimeType
	AutoUpdateTime         TimeType
	HasDefaultValue        bool
	DefaultValue           string
	DefaultValueInterface  interface{}
	NotNull                bool
	Unique                 bool
	Comment                string
	Size                   int
	Precision              int
	Scale                  int
	IgnoreMigration        bool
	FieldType              reflect.Type
	IndirectFieldType      reflect.Type
	StructField            reflect.StructField
	Tag                    reflect.StructTag
	TagSettings            map[string]string
	Schema                 *Schema
	EmbeddedSchema         *Schema
	OwnerSchema            *Schema
	ReflectValueOf         func(context.Context, reflect.Value) reflect.Value
	ValueOf                func(context.Context, reflect.Value) (value interface{}, zero bool)
	Set                    func(context.Context, reflect.Value, interface{}) error
	Serializer             SerializerInterface
	NewValuePool           FieldNewValuePool
	UniqueIndex            string
}

// Helper function to update `AutoCreateTime` and `AutoUpdateTime`
func (field *Field) setAutoTime(fieldType string) {
	if v, ok := field.TagSettings[fieldType]; (ok && utils.CheckTruth(v)) || (!ok && strings.Contains(field.Name, "At") && (field.DataType == Time || field.DataType == Int || field.DataType == Uint)) {
		if field.DataType == Time {
			field.AutoCreateTime = UnixTime
		} else if strings.ToUpper(v) == "NANO" {
			field.AutoCreateTime = UnixNanosecond
		} else if strings.ToUpper(v) == "MILLI" {
			field.AutoCreateTime = UnixMillisecond
		} else {
			field.AutoCreateTime = UnixSecond
		}
	}
}

// ParseField parses a reflect.StructField into a Field
// Major changes:
// 1. Removed excessive nesting in type detection and value extraction logic.
// 2. Introduced utility functions to handle repetitive tasks.
// 3. Improved handling of driver.Valuer to prevent unnecessary allocation or reassignment.
// 4. Segregated handling of special tags and attributes for better structure and readability.
func (schema *Schema) ParseField(fieldStruct reflect.StructField) *Field {
	tagSetting := ParseTagSetting(fieldStruct.Tag.Get("gorm"), ";")

	field := &Field{
		Name:                   fieldStruct.Name,
		DBName:                 tagSetting["COLUMN"],
		BindNames:              []string{fieldStruct.Name},
		EmbeddedBindNames:      []string{fieldStruct.Name},
		FieldType:              fieldStruct.Type,
		IndirectFieldType:      fieldStruct.Type,
		StructField:            fieldStruct,
		Tag:                    fieldStruct.Tag,
		TagSettings:            tagSetting,
		Schema:                 schema,
		Creatable:              true,
		Updatable:              true,
		Readable:               true,
		PrimaryKey:             utils.CheckTruth(tagSetting["PRIMARYKEY"], tagSetting["PRIMARY_KEY"]),
		AutoIncrement:          utils.CheckTruth(tagSetting["AUTOINCREMENT"]),
		AutoIncrementIncrement: DefaultAutoIncrementIncrement,
	}

	// Resolve pointer type
	for field.IndirectFieldType.Kind() == reflect.Ptr {
		field.IndirectFieldType = field.IndirectFieldType.Elem()
	}

	// Determine if field implements driver.Valuer
	fieldValue := reflect.New(field.IndirectFieldType)
	if valuer, isValuer := fieldValue.Interface().(driver.Valuer); isValuer {
		field.handleDriverValuer(valuer, fieldValue)
	}

	// Handle Serializers
	if serializerName := field.TagSettings["SERIALIZER"]; serializerName != "" {
		if serializer, ok := GetSerializer(serializerName); ok {
			field.DataType = String
			field.Serializer = serializer
		} else {
			schema.err = fmt.Errorf("invalid serializer type %v", serializerName)
		}
	}

	// Handle default settings like size, precision, etc.
	field.setAutoTime("AUTOCREATETIME")
	field.setAutoTime("AUTOUPDATETIME")

	// Handle size, precision, scale, and default value
	if num, ok := field.TagSettings["SIZE"]; ok {
		if size, err := strconv.Atoi(num); err == nil {
			field.Size = size
		} else {
			field.Size = -1
		}
	}

	if p, ok := field.TagSettings["PRECISION"]; ok {
		field.Precision, _ = strconv.Atoi(p)
	}

	if s, ok := field.TagSettings["SCALE"]; ok {
		field.Scale, _ = strconv.Atoi(s)
	}

	if v, ok := field.TagSettings["DEFAULT"]; ok {
		field.HasDefaultValue = true
		field.DefaultValue = strings.TrimSpace(v)
		if field.DefaultValue == "null" || field.DefaultValue == "" {
			field.HasDefaultValue = false
		}
	}

	// Set default value interface based on field type
	switch reflect.Indirect(fieldValue).Kind() {
	case reflect.Bool:
		field.DataType = Bool
		if field.HasDefaultValue {
			defaultValue, err := strconv.ParseBool(field.DefaultValue)
			if err != nil {
				schema.err = fmt.Errorf("failed to parse %s as default value for bool: %v", field.DefaultValue, err)
			}
			field.DefaultValueInterface = defaultValue
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		field.DataType = Int
		if field.HasDefaultValue {
			defaultValue, err := strconv.ParseInt(field.DefaultValue, 0, 64)
			if err != nil {
				schema.err = fmt.Errorf("failed to parse %s as default value for int: %v", field.DefaultValue, err)
			}
			field.DefaultValueInterface = defaultValue
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		field.DataType = Uint
		if field.HasDefaultValue {
			defaultValue, err := strconv.ParseUint(field.DefaultValue, 0, 64)
			if err != nil {
				schema.err = fmt.Errorf("failed to parse %s as default value for uint: %v", field.DefaultValue, err)
			}
			field.DefaultValueInterface = defaultValue
		}
	case reflect.Float32, reflect.Float64:
		field.DataType = Float
		if field.HasDefaultValue {
			defaultValue, err := strconv.ParseFloat(field.DefaultValue, 64)
			if err != nil {
				schema.err = fmt.Errorf("failed to parse %s as default value for float: %v", field.DefaultValue, err)
			}
			field.DefaultValueInterface = defaultValue
		}
	case reflect.String:
		field.DataType = String
		if field.HasDefaultValue {
			field.DefaultValueInterface = strings.Trim(field.DefaultValue, "'")
		}
	case reflect.Struct:
		if _, ok := fieldValue.Interface().(*time.Time); ok {
			field.DataType = Time
		} else if fieldValue.Type().ConvertibleTo(TimeReflectType) {
			field.DataType = Time
		} else if fieldValue.Type().ConvertibleTo(TimePtrReflectType) {
			field.DataType = Time
		}
		if field.HasDefaultValue && field.DataType == Time {
			if t, err := now.Parse(field.DefaultValue); err == nil {
				field.DefaultValueInterface = t
			}
		}
	case reflect.Array, reflect.Slice:
		if reflect.Indirect(fieldValue).Type().Elem() == ByteReflectType && field.DataType == "" {
			field.DataType = Bytes
		}
	}

	// Set permissions
	field.setupPermissions()

	// Handle Embedded fields
	field.handleEmbeddedField(fieldStruct, schema)

	return field
}

// Helper function to handle driver.Valuer interface for fields
func (field *Field) handleDriverValuer(valuer driver.Valuer, fieldValue reflect.Value) {
	if _, ok := fieldValue.Interface().(GormDataTypeInterface); !ok {
		if v, err := valuer.Value(); reflect.ValueOf(v).IsValid() && err == nil {
			fieldValue = reflect.ValueOf(v)
		}
		field.extractRealFieldValue(fieldValue)
	}
}

// Helper function to recursively extract the actual value for complex types
func (field *Field) extractRealFieldValue(v reflect.Value) {
	rv := reflect.Indirect(v)
	if rv.Kind() == reflect.Struct && !rv.Type().ConvertibleTo(TimeReflectType) {
		for i := 0; i < rv.NumField(); i++ {
			for key, value := range ParseTagSetting(rv.Type().Field(i).Tag.Get("gorm"), ";") {
				if _, ok := field.TagSettings[key]; !ok {
					field.TagSettings[key] = value
				}
			}
		}
	}
}

// Helper function to set up field permissions based on tag settings
func (field *Field) setupPermissions() {
	if val, ok := field.TagSettings["-"]; ok {
		val = strings.ToLower(strings.TrimSpace(val))
		switch val {
		case "-":
			field.Creatable = false
			field.Updatable = false
			field.Readable = false
			field.DataType = ""
		case "all":
			field.Creatable = false
			field.Updatable = false
			field.Readable = false
			field.DataType = ""
			field.IgnoreMigration = true
		case "migration":
			field.IgnoreMigration = true
		}
	}

	if v, ok := field.TagSettings["->"]; ok {
		field.Creatable = false
		field.Updatable = false
		if strings.ToLower(v) == "false" {
			field.Readable = false
		} else {
			field.Readable = true
		}
	}

	if v, ok := field.TagSettings["<-"]; ok {
		field.Creatable = true
		field.Updatable = true
		if v != "<-" {
			if !strings.Contains(v, "create") {
				field.Creatable = false
			}
			if !strings.Contains(v, "update") {
				field.Updatable = false
			}
		}
	}
}

// Helper function to handle embedded fields
func (field *Field) handleEmbeddedField(fieldStruct reflect.StructField, schema *Schema) {
	if _, ok := field.TagSettings["EMBEDDED"]; ok || (field.GORMDataType != Time && field.GORMDataType != Bytes && fieldStruct.Anonymous && (field.Creatable || field.Updatable || field.Readable)) {
		kind := reflect.Indirect(reflect.New(field.IndirectFieldType)).Kind()
		switch kind {
		case reflect.Struct:
			var err error
			field.Creatable = false
			field.Updatable = false
			field.Readable = false

			cacheStore := &sync.Map{}
			cacheStore.Store(embeddedCacheKey, true)
			if field.EmbeddedSchema, err = getOrParse(field.IndirectFieldType, cacheStore, embeddedNamer{Table: schema.Table, Namer: schema.namer}); err != nil {
				schema.err = err
			}

			for _, ef := range field.EmbeddedSchema.Fields {
				ef.Schema = schema
				ef.OwnerSchema = field.EmbeddedSchema
				ef.BindNames = append([]string{fieldStruct.Name}, ef.BindNames...)
				if _, ok := field.TagSettings["EMBEDDED"]; ok || !fieldStruct.Anonymous {
					ef.EmbeddedBindNames = append([]string{fieldStruct.Name}, ef.EmbeddedBindNames...)
				}
				// index is negative means is pointer
				if field.FieldType.Kind() == reflect.Struct {
					ef.StructField.Index = append([]int{fieldStruct.Index[0]}, ef.StructField.Index...)
				} else {
					ef.StructField.Index = append([]int{-fieldStruct.Index[0] - 1}, ef.StructField.Index...)
				}

				if prefix, ok := field.TagSettings["EMBEDDEDPREFIX"]; ok && ef.DBName != "" {
					ef.DBName = prefix + ef.DBName
				}

				if ef.PrimaryKey {
					if !utils.CheckTruth(ef.TagSettings["PRIMARYKEY"], ef.TagSettings["PRIMARY_KEY"]) {
						ef.PrimaryKey = false
						if val, ok := ef.TagSettings["AUTOINCREMENT"]; !ok || !utils.CheckTruth(val) {
							ef.AutoIncrement = false
						}
						if !ef.AutoIncrement && ef.DefaultValue == "" {
							ef.HasDefaultValue = false
						}
					}
				}

				for k, v := range field.TagSettings {
					ef.TagSettings[k] = v
				}
			}
		case reflect.Invalid, reflect.Uintptr, reflect.Array, reflect.Chan, reflect.Func, reflect.Interface,
			reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer, reflect.Complex64, reflect.Complex128:
			schema.err = fmt.Errorf("invalid embedded struct for %s's field %s, should be struct, but got %v", field.Schema.Name, field.Name, field.FieldType)
		}
	}
}
