package schema

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jinzhu/now"
	"gorm.io/gorm/utils"
)

type DataType string

type TimeType int64

var TimeReflectType = reflect.TypeOf(time.Time{})

const (
	UnixSecond      TimeType = 1
	UnixMillisecond TimeType = 2
	UnixNanosecond  TimeType = 3
)

const (
	Bool   DataType = "bool"
	Int    DataType = "int"
	Uint   DataType = "uint"
	Float  DataType = "float"
	String DataType = "string"
	Time   DataType = "time"
	Bytes  DataType = "bytes"
)

type Field struct {
	Name                  string
	DBName                string
	BindNames             []string
	DataType              DataType
	GORMDataType          DataType
	PrimaryKey            bool
	AutoIncrement         bool
	Creatable             bool
	Updatable             bool
	Readable              bool
	HasDefaultValue       bool
	AutoCreateTime        TimeType
	AutoUpdateTime        TimeType
	DefaultValue          string
	DefaultValueInterface interface{}
	NotNull               bool
	Unique                bool
	Comment               string
	Size                  int
	Precision             int
	Scale                 int
	FieldType             reflect.Type
	IndirectFieldType     reflect.Type
	StructField           reflect.StructField
	Tag                   reflect.StructTag
	TagSettings           map[string]string
	Schema                *Schema
	EmbeddedSchema        *Schema
	OwnerSchema           *Schema
	ReflectValueOf        func(reflect.Value) reflect.Value
	ValueOf               func(reflect.Value) (value interface{}, zero bool)
	Set                   func(reflect.Value, interface{}) error
}

func (schema *Schema) ParseField(fieldStruct reflect.StructField) *Field {
	var err error

	field := &Field{
		Name:              fieldStruct.Name,
		BindNames:         []string{fieldStruct.Name},
		FieldType:         fieldStruct.Type,
		IndirectFieldType: fieldStruct.Type,
		StructField:       fieldStruct,
		Creatable:         true,
		Updatable:         true,
		Readable:          true,
		Tag:               fieldStruct.Tag,
		TagSettings:       ParseTagSetting(fieldStruct.Tag.Get("gorm"), ";"),
		Schema:            schema,
	}

	for field.IndirectFieldType.Kind() == reflect.Ptr {
		field.IndirectFieldType = field.IndirectFieldType.Elem()
	}

	fieldValue := reflect.New(field.IndirectFieldType)
	// if field is valuer, used its value or first fields as data type
	valuer, isValuer := fieldValue.Interface().(driver.Valuer)
	if isValuer {
		if _, ok := fieldValue.Interface().(GormDataTypeInterface); !ok {
			if v, err := valuer.Value(); reflect.ValueOf(v).IsValid() && err == nil {
				fieldValue = reflect.ValueOf(v)
			}

			var getRealFieldValue func(reflect.Value)
			getRealFieldValue = func(v reflect.Value) {
				rv := reflect.Indirect(v)
				if rv.Kind() == reflect.Struct && !rv.Type().ConvertibleTo(TimeReflectType) {
					for i := 0; i < rv.Type().NumField(); i++ {
						newFieldType := rv.Type().Field(i).Type
						for newFieldType.Kind() == reflect.Ptr {
							newFieldType = newFieldType.Elem()
						}

						fieldValue = reflect.New(newFieldType)

						if rv.Type() != reflect.Indirect(fieldValue).Type() {
							getRealFieldValue(fieldValue)
						}

						if fieldValue.IsValid() {
							return
						}

						for key, value := range ParseTagSetting(field.IndirectFieldType.Field(i).Tag.Get("gorm"), ";") {
							if _, ok := field.TagSettings[key]; !ok {
								field.TagSettings[key] = value
							}
						}
					}
				}
			}

			getRealFieldValue(fieldValue)
		}
	}

	if dbName, ok := field.TagSettings["COLUMN"]; ok {
		field.DBName = dbName
	}

	if val, ok := field.TagSettings["PRIMARYKEY"]; ok && utils.CheckTruth(val) {
		field.PrimaryKey = true
	} else if val, ok := field.TagSettings["PRIMARY_KEY"]; ok && utils.CheckTruth(val) {
		field.PrimaryKey = true
	}

	if val, ok := field.TagSettings["AUTOINCREMENT"]; ok && utils.CheckTruth(val) {
		field.AutoIncrement = true
		field.HasDefaultValue = true
	}

	if v, ok := field.TagSettings["DEFAULT"]; ok {
		field.HasDefaultValue = true
		field.DefaultValue = v
	}

	if num, ok := field.TagSettings["SIZE"]; ok {
		if field.Size, err = strconv.Atoi(num); err != nil {
			field.Size = -1
		}
	}

	if p, ok := field.TagSettings["PRECISION"]; ok {
		field.Precision, _ = strconv.Atoi(p)
	}

	if s, ok := field.TagSettings["SCALE"]; ok {
		field.Scale, _ = strconv.Atoi(s)
	}

	if val, ok := field.TagSettings["NOT NULL"]; ok && utils.CheckTruth(val) {
		field.NotNull = true
	}

	if val, ok := field.TagSettings["UNIQUE"]; ok && utils.CheckTruth(val) {
		field.Unique = true
	}

	if val, ok := field.TagSettings["COMMENT"]; ok {
		field.Comment = val
	}

	// default value is function or null or blank (primary keys)
	skipParseDefaultValue := strings.Contains(field.DefaultValue, "(") &&
		strings.Contains(field.DefaultValue, ")") || strings.ToLower(field.DefaultValue) == "null" || field.DefaultValue == ""
	switch reflect.Indirect(fieldValue).Kind() {
	case reflect.Bool:
		field.DataType = Bool
		if field.HasDefaultValue && !skipParseDefaultValue {
			if field.DefaultValueInterface, err = strconv.ParseBool(field.DefaultValue); err != nil {
				schema.err = fmt.Errorf("failed to parse %v as default value for bool, got error: %v", field.DefaultValue, err)
			}
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		field.DataType = Int
		if field.HasDefaultValue && !skipParseDefaultValue {
			if field.DefaultValueInterface, err = strconv.ParseInt(field.DefaultValue, 0, 64); err != nil {
				schema.err = fmt.Errorf("failed to parse %v as default value for int, got error: %v", field.DefaultValue, err)
			}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		field.DataType = Uint
		if field.HasDefaultValue && !skipParseDefaultValue {
			if field.DefaultValueInterface, err = strconv.ParseUint(field.DefaultValue, 0, 64); err != nil {
				schema.err = fmt.Errorf("failed to parse %v as default value for uint, got error: %v", field.DefaultValue, err)
			}
		}
	case reflect.Float32, reflect.Float64:
		field.DataType = Float
		if field.HasDefaultValue && !skipParseDefaultValue {
			if field.DefaultValueInterface, err = strconv.ParseFloat(field.DefaultValue, 64); err != nil {
				schema.err = fmt.Errorf("failed to parse %v as default value for float, got error: %v", field.DefaultValue, err)
			}
		}
	case reflect.String:
		field.DataType = String

		if field.HasDefaultValue && !skipParseDefaultValue {
			field.DefaultValue = strings.Trim(field.DefaultValue, "'")
			field.DefaultValue = strings.Trim(field.DefaultValue, "\"")
			field.DefaultValueInterface = field.DefaultValue
		}
	case reflect.Struct:
		if _, ok := fieldValue.Interface().(*time.Time); ok {
			field.DataType = Time
		} else if fieldValue.Type().ConvertibleTo(TimeReflectType) {
			field.DataType = Time
		} else if fieldValue.Type().ConvertibleTo(reflect.TypeOf(&time.Time{})) {
			field.DataType = Time
		}
	case reflect.Array, reflect.Slice:
		if reflect.Indirect(fieldValue).Type().Elem() == reflect.TypeOf(uint8(0)) {
			field.DataType = Bytes
		}
	}

	field.GORMDataType = field.DataType

	if dataTyper, ok := fieldValue.Interface().(GormDataTypeInterface); ok {
		field.DataType = DataType(dataTyper.GormDataType())
	}

	if v, ok := field.TagSettings["AUTOCREATETIME"]; ok || (field.Name == "CreatedAt" && (field.DataType == Time || field.DataType == Int || field.DataType == Uint)) {
		if strings.ToUpper(v) == "NANO" {
			field.AutoCreateTime = UnixNanosecond
		} else if strings.ToUpper(v) == "MILLI" {
			field.AutoCreateTime = UnixMillisecond
		} else {
			field.AutoCreateTime = UnixSecond
		}
	}

	if v, ok := field.TagSettings["AUTOUPDATETIME"]; ok || (field.Name == "UpdatedAt" && (field.DataType == Time || field.DataType == Int || field.DataType == Uint)) {
		if strings.ToUpper(v) == "NANO" {
			field.AutoUpdateTime = UnixNanosecond
		} else if strings.ToUpper(v) == "MILLI" {
			field.AutoUpdateTime = UnixMillisecond
		} else {
			field.AutoUpdateTime = UnixSecond
		}
	}

	if val, ok := field.TagSettings["TYPE"]; ok {
		switch DataType(strings.ToLower(val)) {
		case Bool, Int, Uint, Float, String, Time, Bytes:
			field.DataType = DataType(strings.ToLower(val))
		default:
			field.DataType = DataType(val)
		}
	}

	if field.GORMDataType == "" {
		field.GORMDataType = field.DataType
	}

	if field.Size == 0 {
		switch reflect.Indirect(fieldValue).Kind() {
		case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint64, reflect.Float64:
			field.Size = 64
		case reflect.Int8, reflect.Uint8:
			field.Size = 8
		case reflect.Int16, reflect.Uint16:
			field.Size = 16
		case reflect.Int32, reflect.Uint32, reflect.Float32:
			field.Size = 32
		}
	}

	// setup permission
	if _, ok := field.TagSettings["-"]; ok {
		field.Creatable = false
		field.Updatable = false
		field.Readable = false
		field.DataType = ""
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

	if _, ok := field.TagSettings["EMBEDDED"]; ok || (fieldStruct.Anonymous && !isValuer && (field.Creatable || field.Updatable || field.Readable)) {
		if reflect.Indirect(fieldValue).Kind() == reflect.Struct {
			var err error
			field.Creatable = false
			field.Updatable = false
			field.Readable = false

			cacheStore := &sync.Map{}
			cacheStore.Store(embeddedCacheKey, true)
			if field.EmbeddedSchema, err = Parse(fieldValue.Interface(), cacheStore, embeddedNamer{Table: schema.Table, Namer: schema.namer}); err != nil {
				schema.err = err
			}

			for _, ef := range field.EmbeddedSchema.Fields {
				ef.Schema = schema
				ef.OwnerSchema = field.EmbeddedSchema
				ef.BindNames = append([]string{fieldStruct.Name}, ef.BindNames...)
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
					if val, ok := ef.TagSettings["PRIMARYKEY"]; ok && utils.CheckTruth(val) {
						ef.PrimaryKey = true
					} else if val, ok := ef.TagSettings["PRIMARY_KEY"]; ok && utils.CheckTruth(val) {
						ef.PrimaryKey = true
					} else {
						ef.PrimaryKey = false

						if val, ok := ef.TagSettings["AUTOINCREMENT"]; !ok || !utils.CheckTruth(val) {
							ef.AutoIncrement = false
						}

						if ef.DefaultValue == "" {
							ef.HasDefaultValue = false
						}
					}
				}

				for k, v := range field.TagSettings {
					ef.TagSettings[k] = v
				}
			}
		} else {
			schema.err = fmt.Errorf("invalid embedded struct for %v's field %v, should be struct, but got %v", field.Schema.Name, field.Name, field.FieldType)
		}
	}

	return field
}

// create valuer, setter when parse struct
func (field *Field) setupValuerAndSetter() {
	// ValueOf
	switch {
	case len(field.StructField.Index) == 1:
		field.ValueOf = func(value reflect.Value) (interface{}, bool) {
			fieldValue := reflect.Indirect(value).Field(field.StructField.Index[0])
			return fieldValue.Interface(), fieldValue.IsZero()
		}
	case len(field.StructField.Index) == 2 && field.StructField.Index[0] >= 0:
		field.ValueOf = func(value reflect.Value) (interface{}, bool) {
			fieldValue := reflect.Indirect(value).Field(field.StructField.Index[0]).Field(field.StructField.Index[1])
			return fieldValue.Interface(), fieldValue.IsZero()
		}
	default:
		field.ValueOf = func(value reflect.Value) (interface{}, bool) {
			v := reflect.Indirect(value)

			for _, idx := range field.StructField.Index {
				if idx >= 0 {
					v = v.Field(idx)
				} else {
					v = v.Field(-idx - 1)

					if v.Type().Elem().Kind() == reflect.Struct {
						if !v.IsNil() {
							v = v.Elem()
						} else {
							return nil, true
						}
					} else {
						return nil, true
					}
				}
			}
			return v.Interface(), v.IsZero()
		}
	}

	// ReflectValueOf
	switch {
	case len(field.StructField.Index) == 1:
		if field.FieldType.Kind() == reflect.Ptr {
			field.ReflectValueOf = func(value reflect.Value) reflect.Value {
				fieldValue := reflect.Indirect(value).Field(field.StructField.Index[0])
				return fieldValue
			}
		} else {
			field.ReflectValueOf = func(value reflect.Value) reflect.Value {
				return reflect.Indirect(value).Field(field.StructField.Index[0])
			}
		}
	case len(field.StructField.Index) == 2 && field.StructField.Index[0] >= 0 && field.FieldType.Kind() != reflect.Ptr:
		field.ReflectValueOf = func(value reflect.Value) reflect.Value {
			return reflect.Indirect(value).Field(field.StructField.Index[0]).Field(field.StructField.Index[1])
		}
	default:
		field.ReflectValueOf = func(value reflect.Value) reflect.Value {
			v := reflect.Indirect(value)
			for idx, fieldIdx := range field.StructField.Index {
				if fieldIdx >= 0 {
					v = v.Field(fieldIdx)
				} else {
					v = v.Field(-fieldIdx - 1)
				}

				if v.Kind() == reflect.Ptr {
					if v.Type().Elem().Kind() == reflect.Struct {
						if v.IsNil() {
							v.Set(reflect.New(v.Type().Elem()))
						}
					}

					if idx < len(field.StructField.Index)-1 {
						v = v.Elem()
					}
				}
			}
			return v
		}
	}

	fallbackSetter := func(value reflect.Value, v interface{}, setter func(reflect.Value, interface{}) error) (err error) {
		if v == nil {
			field.ReflectValueOf(value).Set(reflect.New(field.FieldType).Elem())
		} else {
			reflectV := reflect.ValueOf(v)

			if reflectV.Type().AssignableTo(field.FieldType) {
				field.ReflectValueOf(value).Set(reflectV)
				return
			} else if reflectV.Type().ConvertibleTo(field.FieldType) {
				field.ReflectValueOf(value).Set(reflectV.Convert(field.FieldType))
				return
			} else if field.FieldType.Kind() == reflect.Ptr {
				fieldValue := field.ReflectValueOf(value)

				if reflectV.Type().AssignableTo(field.FieldType.Elem()) {
					if !fieldValue.IsValid() {
						fieldValue = reflect.New(field.FieldType.Elem())
					} else if fieldValue.IsNil() {
						fieldValue.Set(reflect.New(field.FieldType.Elem()))
					}
					fieldValue.Elem().Set(reflectV)
					return
				} else if reflectV.Type().ConvertibleTo(field.FieldType.Elem()) {
					if fieldValue.IsNil() {
						fieldValue.Set(reflect.New(field.FieldType.Elem()))
					}

					fieldValue.Elem().Set(reflectV.Convert(field.FieldType.Elem()))
					return
				}
			}

			if reflectV.Kind() == reflect.Ptr {
				if reflectV.IsNil() {
					field.ReflectValueOf(value).Set(reflect.New(field.FieldType).Elem())
				} else {
					err = setter(value, reflectV.Elem().Interface())
				}
			} else if valuer, ok := v.(driver.Valuer); ok {
				if v, err = valuer.Value(); err == nil {
					err = setter(value, v)
				}
			} else {
				return fmt.Errorf("failed to set value %+v to field %v", v, field.Name)
			}
		}

		return
	}

	// Set
	switch field.FieldType.Kind() {
	case reflect.Bool:
		field.Set = func(value reflect.Value, v interface{}) error {
			switch data := v.(type) {
			case bool:
				field.ReflectValueOf(value).SetBool(data)
			case *bool:
				if data != nil {
					field.ReflectValueOf(value).SetBool(*data)
				} else {
					field.ReflectValueOf(value).SetBool(false)
				}
			case int64:
				if data > 0 {
					field.ReflectValueOf(value).SetBool(true)
				} else {
					field.ReflectValueOf(value).SetBool(false)
				}
			case string:
				b, _ := strconv.ParseBool(data)
				field.ReflectValueOf(value).SetBool(b)
			default:
				return fallbackSetter(value, v, field.Set)
			}
			return nil
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		field.Set = func(value reflect.Value, v interface{}) (err error) {
			switch data := v.(type) {
			case int64:
				field.ReflectValueOf(value).SetInt(data)
			case int:
				field.ReflectValueOf(value).SetInt(int64(data))
			case int8:
				field.ReflectValueOf(value).SetInt(int64(data))
			case int16:
				field.ReflectValueOf(value).SetInt(int64(data))
			case int32:
				field.ReflectValueOf(value).SetInt(int64(data))
			case uint:
				field.ReflectValueOf(value).SetInt(int64(data))
			case uint8:
				field.ReflectValueOf(value).SetInt(int64(data))
			case uint16:
				field.ReflectValueOf(value).SetInt(int64(data))
			case uint32:
				field.ReflectValueOf(value).SetInt(int64(data))
			case uint64:
				field.ReflectValueOf(value).SetInt(int64(data))
			case float32:
				field.ReflectValueOf(value).SetInt(int64(data))
			case float64:
				field.ReflectValueOf(value).SetInt(int64(data))
			case []byte:
				return field.Set(value, string(data))
			case string:
				if i, err := strconv.ParseInt(data, 0, 64); err == nil {
					field.ReflectValueOf(value).SetInt(i)
				} else {
					return err
				}
			case time.Time:
				if field.AutoCreateTime == UnixNanosecond || field.AutoUpdateTime == UnixNanosecond {
					field.ReflectValueOf(value).SetInt(data.UnixNano())
				} else if field.AutoCreateTime == UnixMillisecond || field.AutoUpdateTime == UnixMillisecond {
					field.ReflectValueOf(value).SetInt(data.UnixNano() / 1e6)
				} else {
					field.ReflectValueOf(value).SetInt(data.Unix())
				}
			case *time.Time:
				if data != nil {
					if field.AutoCreateTime == UnixNanosecond || field.AutoUpdateTime == UnixNanosecond {
						field.ReflectValueOf(value).SetInt(data.UnixNano())
					} else if field.AutoCreateTime == UnixMillisecond || field.AutoUpdateTime == UnixMillisecond {
						field.ReflectValueOf(value).SetInt(data.UnixNano() / 1e6)
					} else {
						field.ReflectValueOf(value).SetInt(data.Unix())
					}
				} else {
					field.ReflectValueOf(value).SetInt(0)
				}
			default:
				return fallbackSetter(value, v, field.Set)
			}
			return err
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		field.Set = func(value reflect.Value, v interface{}) (err error) {
			switch data := v.(type) {
			case uint64:
				field.ReflectValueOf(value).SetUint(data)
			case uint:
				field.ReflectValueOf(value).SetUint(uint64(data))
			case uint8:
				field.ReflectValueOf(value).SetUint(uint64(data))
			case uint16:
				field.ReflectValueOf(value).SetUint(uint64(data))
			case uint32:
				field.ReflectValueOf(value).SetUint(uint64(data))
			case int64:
				field.ReflectValueOf(value).SetUint(uint64(data))
			case int:
				field.ReflectValueOf(value).SetUint(uint64(data))
			case int8:
				field.ReflectValueOf(value).SetUint(uint64(data))
			case int16:
				field.ReflectValueOf(value).SetUint(uint64(data))
			case int32:
				field.ReflectValueOf(value).SetUint(uint64(data))
			case float32:
				field.ReflectValueOf(value).SetUint(uint64(data))
			case float64:
				field.ReflectValueOf(value).SetUint(uint64(data))
			case []byte:
				return field.Set(value, string(data))
			case time.Time:
				if field.AutoCreateTime == UnixNanosecond || field.AutoUpdateTime == UnixNanosecond {
					field.ReflectValueOf(value).SetUint(uint64(data.UnixNano()))
				} else if field.AutoCreateTime == UnixMillisecond || field.AutoUpdateTime == UnixMillisecond {
					field.ReflectValueOf(value).SetUint(uint64(data.UnixNano() / 1e6))
				} else {
					field.ReflectValueOf(value).SetUint(uint64(data.Unix()))
				}
			case string:
				if i, err := strconv.ParseUint(data, 0, 64); err == nil {
					field.ReflectValueOf(value).SetUint(i)
				} else {
					return err
				}
			default:
				return fallbackSetter(value, v, field.Set)
			}
			return err
		}
	case reflect.Float32, reflect.Float64:
		field.Set = func(value reflect.Value, v interface{}) (err error) {
			switch data := v.(type) {
			case float64:
				field.ReflectValueOf(value).SetFloat(data)
			case float32:
				field.ReflectValueOf(value).SetFloat(float64(data))
			case int64:
				field.ReflectValueOf(value).SetFloat(float64(data))
			case int:
				field.ReflectValueOf(value).SetFloat(float64(data))
			case int8:
				field.ReflectValueOf(value).SetFloat(float64(data))
			case int16:
				field.ReflectValueOf(value).SetFloat(float64(data))
			case int32:
				field.ReflectValueOf(value).SetFloat(float64(data))
			case uint:
				field.ReflectValueOf(value).SetFloat(float64(data))
			case uint8:
				field.ReflectValueOf(value).SetFloat(float64(data))
			case uint16:
				field.ReflectValueOf(value).SetFloat(float64(data))
			case uint32:
				field.ReflectValueOf(value).SetFloat(float64(data))
			case uint64:
				field.ReflectValueOf(value).SetFloat(float64(data))
			case []byte:
				return field.Set(value, string(data))
			case string:
				if i, err := strconv.ParseFloat(data, 64); err == nil {
					field.ReflectValueOf(value).SetFloat(i)
				} else {
					return err
				}
			default:
				return fallbackSetter(value, v, field.Set)
			}
			return err
		}
	case reflect.String:
		field.Set = func(value reflect.Value, v interface{}) (err error) {
			switch data := v.(type) {
			case string:
				field.ReflectValueOf(value).SetString(data)
			case []byte:
				field.ReflectValueOf(value).SetString(string(data))
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				field.ReflectValueOf(value).SetString(utils.ToString(data))
			case float64, float32:
				field.ReflectValueOf(value).SetString(fmt.Sprintf("%."+strconv.Itoa(field.Precision)+"f", data))
			default:
				return fallbackSetter(value, v, field.Set)
			}
			return err
		}
	default:
		fieldValue := reflect.New(field.FieldType)
		switch fieldValue.Elem().Interface().(type) {
		case time.Time:
			field.Set = func(value reflect.Value, v interface{}) error {
				switch data := v.(type) {
				case time.Time:
					field.ReflectValueOf(value).Set(reflect.ValueOf(v))
				case *time.Time:
					if data != nil {
						field.ReflectValueOf(value).Set(reflect.ValueOf(data).Elem())
					} else {
						field.ReflectValueOf(value).Set(reflect.ValueOf(time.Time{}))
					}
				case string:
					if t, err := now.Parse(data); err == nil {
						field.ReflectValueOf(value).Set(reflect.ValueOf(t))
					} else {
						return fmt.Errorf("failed to set string %v to time.Time field %v, failed to parse it as time, got error %v", v, field.Name, err)
					}
				default:
					return fallbackSetter(value, v, field.Set)
				}
				return nil
			}
		case *time.Time:
			field.Set = func(value reflect.Value, v interface{}) error {
				switch data := v.(type) {
				case time.Time:
					fieldValue := field.ReflectValueOf(value)
					if fieldValue.IsNil() {
						fieldValue.Set(reflect.New(field.FieldType.Elem()))
					}
					fieldValue.Elem().Set(reflect.ValueOf(v))
				case *time.Time:
					field.ReflectValueOf(value).Set(reflect.ValueOf(v))
				case string:
					if t, err := now.Parse(data); err == nil {
						fieldValue := field.ReflectValueOf(value)
						if fieldValue.IsNil() {
							if v == "" {
								return nil
							}
							fieldValue.Set(reflect.New(field.FieldType.Elem()))
						}
						fieldValue.Elem().Set(reflect.ValueOf(t))
					} else {
						return fmt.Errorf("failed to set string %v to time.Time field %v, failed to parse it as time, got error %v", v, field.Name, err)
					}
				default:
					return fallbackSetter(value, v, field.Set)
				}
				return nil
			}
		default:
			if _, ok := fieldValue.Elem().Interface().(sql.Scanner); ok {
				// pointer scanner
				field.Set = func(value reflect.Value, v interface{}) (err error) {
					reflectV := reflect.ValueOf(v)
					if reflectV.Type().AssignableTo(field.FieldType) {
						field.ReflectValueOf(value).Set(reflectV)
					} else if reflectV.Kind() == reflect.Ptr {
						if reflectV.IsNil() {
							field.ReflectValueOf(value).Set(reflect.New(field.FieldType).Elem())
						} else {
							err = field.Set(value, reflectV.Elem().Interface())
						}
					} else {
						fieldValue := field.ReflectValueOf(value)
						if fieldValue.IsNil() {
							fieldValue.Set(reflect.New(field.FieldType.Elem()))
						}

						if valuer, ok := v.(driver.Valuer); ok {
							v, _ = valuer.Value()
						}

						err = fieldValue.Interface().(sql.Scanner).Scan(v)
					}
					return
				}
			} else if _, ok := fieldValue.Interface().(sql.Scanner); ok {
				// struct scanner
				field.Set = func(value reflect.Value, v interface{}) (err error) {
					reflectV := reflect.ValueOf(v)
					if !reflectV.IsValid() {
						field.ReflectValueOf(value).Set(reflect.New(field.FieldType).Elem())
					} else if reflectV.Type().AssignableTo(field.FieldType) {
						field.ReflectValueOf(value).Set(reflectV)
					} else if reflectV.Kind() == reflect.Ptr {
						if reflectV.IsNil() || !reflectV.IsValid() {
							field.ReflectValueOf(value).Set(reflect.New(field.FieldType).Elem())
						} else {
							return field.Set(value, reflectV.Elem().Interface())
						}
					} else {
						if valuer, ok := v.(driver.Valuer); ok {
							v, _ = valuer.Value()
						}

						err = field.ReflectValueOf(value).Addr().Interface().(sql.Scanner).Scan(v)
					}
					return
				}
			} else {
				field.Set = func(value reflect.Value, v interface{}) (err error) {
					return fallbackSetter(value, v, field.Set)
				}
			}
		}
	}
}
