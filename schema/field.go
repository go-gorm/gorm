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
)

type DataType string

type TimeType int64

const (
	UnixSecond     TimeType = 1
	UnixNanosecond TimeType = 2
)

const (
	Bool   DataType = "bool"
	Int             = "int"
	Uint            = "uint"
	Float           = "float"
	String          = "string"
	Time            = "time"
	Bytes           = "bytes"
)

type Field struct {
	Name                  string
	DBName                string
	BindNames             []string
	DataType              DataType
	DBDataType            string
	PrimaryKey            bool
	AutoIncrement         bool
	Creatable             bool
	Updatable             bool
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
	FieldType             reflect.Type
	IndirectFieldType     reflect.Type
	StructField           reflect.StructField
	Tag                   reflect.StructTag
	TagSettings           map[string]string
	Schema                *Schema
	EmbeddedSchema        *Schema
	ReflectValueOf        func(reflect.Value) reflect.Value
	ValueOf               func(reflect.Value) (value interface{}, zero bool)
	Set                   func(reflect.Value, interface{}) error
}

func (schema *Schema) ParseField(fieldStruct reflect.StructField) *Field {
	field := &Field{
		Name:              fieldStruct.Name,
		BindNames:         []string{fieldStruct.Name},
		FieldType:         fieldStruct.Type,
		IndirectFieldType: fieldStruct.Type,
		StructField:       fieldStruct,
		Creatable:         true,
		Updatable:         true,
		Tag:               fieldStruct.Tag,
		TagSettings:       ParseTagSetting(fieldStruct.Tag.Get("gorm"), ";"),
		Schema:            schema,
	}

	for field.IndirectFieldType.Kind() == reflect.Ptr {
		field.IndirectFieldType = field.IndirectFieldType.Elem()
	}

	fieldValue := reflect.New(field.IndirectFieldType)
	// if field is valuer, used its value or first fields as data type
	if valuer, isValueOf := fieldValue.Interface().(driver.Valuer); isValueOf {
		var overrideFieldValue bool
		if v, err := valuer.Value(); v != nil && err == nil {
			overrideFieldValue = true
			fieldValue = reflect.ValueOf(v)
		}

		if field.IndirectFieldType.Kind() == reflect.Struct {
			for i := 0; i < field.IndirectFieldType.NumField(); i++ {
				if !overrideFieldValue {
					newFieldType := field.IndirectFieldType.Field(i).Type
					for newFieldType.Kind() == reflect.Ptr {
						newFieldType = newFieldType.Elem()
					}

					fieldValue = reflect.New(newFieldType)
					overrideFieldValue = true
				}

				// copy tag settings from valuer
				for key, value := range ParseTagSetting(field.IndirectFieldType.Field(i).Tag.Get("gorm"), ";") {
					if _, ok := field.TagSettings[key]; !ok {
						field.TagSettings[key] = value
					}
				}
			}
		}
	}

	// setup permission
	if _, ok := field.TagSettings["-"]; ok {
		field.Creatable = false
		field.Updatable = false
	}

	if dbName, ok := field.TagSettings["COLUMN"]; ok {
		field.DBName = dbName
	}

	if val, ok := field.TagSettings["PRIMARYKEY"]; ok && checkTruth(val) {
		field.PrimaryKey = true
	}

	if val, ok := field.TagSettings["AUTOINCREMENT"]; ok && checkTruth(val) {
		field.AutoIncrement = true
		field.HasDefaultValue = true
	}

	if v, ok := field.TagSettings["DEFAULT"]; ok {
		field.HasDefaultValue = true
		field.DefaultValue = v
	}

	if num, ok := field.TagSettings["SIZE"]; ok {
		var err error
		if field.Size, err = strconv.Atoi(num); err != nil {
			field.Size = -1
		}
	}

	if p, ok := field.TagSettings["PRECISION"]; ok {
		field.Precision, _ = strconv.Atoi(p)
	}

	if val, ok := field.TagSettings["NOT NULL"]; ok && checkTruth(val) {
		field.NotNull = true
	}

	if val, ok := field.TagSettings["UNIQUE"]; ok && checkTruth(val) {
		field.Unique = true
	}

	if val, ok := field.TagSettings["COMMENT"]; ok {
		field.Comment = val
	}

	if val, ok := field.TagSettings["TYPE"]; ok {
		field.DBDataType = val
	}

	switch fieldValue.Elem().Kind() {
	case reflect.Bool:
		field.DataType = Bool
		if field.HasDefaultValue {
			field.DefaultValueInterface, _ = strconv.ParseBool(field.DefaultValue)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		field.DataType = Int
		if field.HasDefaultValue {
			field.DefaultValueInterface, _ = strconv.ParseInt(field.DefaultValue, 0, 64)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		field.DataType = Uint
		if field.HasDefaultValue {
			field.DefaultValueInterface, _ = strconv.ParseUint(field.DefaultValue, 0, 64)
		}
	case reflect.Float32, reflect.Float64:
		field.DataType = Float
		if field.HasDefaultValue {
			field.DefaultValueInterface, _ = strconv.ParseFloat(field.DefaultValue, 64)
		}
	case reflect.String:
		field.DataType = String
		if field.HasDefaultValue {
			field.DefaultValueInterface = field.DefaultValue
		}
	case reflect.Struct:
		if _, ok := fieldValue.Interface().(*time.Time); ok {
			field.DataType = Time
		} else if fieldValue.Type().ConvertibleTo(reflect.TypeOf(&time.Time{})) {
			field.DataType = Time
		}
	case reflect.Array, reflect.Slice:
		if fieldValue.Type().Elem() == reflect.TypeOf(uint8(0)) {
			field.DataType = Bytes
		}
	}

	if v, ok := field.TagSettings["AUTOCREATETIME"]; ok || (field.Name == "CreatedAt" && (field.DataType == Time || field.DataType == Int)) {
		if strings.ToUpper(v) == "NANO" {
			field.AutoCreateTime = UnixNanosecond
		} else {
			field.AutoCreateTime = UnixSecond
		}
	}

	if v, ok := field.TagSettings["AUTOUPDATETIME"]; ok || (field.Name == "UpdatedAt" && (field.DataType == Time || field.DataType == Int)) {
		if strings.ToUpper(v) == "NANO" {
			field.AutoUpdateTime = UnixNanosecond
		} else {
			field.AutoUpdateTime = UnixSecond
		}
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

	if _, ok := field.TagSettings["EMBEDDED"]; ok || fieldStruct.Anonymous {
		var err error
		field.Creatable = false
		field.Updatable = false
		if field.EmbeddedSchema, err = Parse(fieldValue.Interface(), &sync.Map{}, schema.namer); err != nil {
			schema.err = err
		}
		for _, ef := range field.EmbeddedSchema.Fields {
			ef.Schema = schema
			ef.BindNames = append([]string{fieldStruct.Name}, ef.BindNames...)
			// index is negative means is pointer
			if field.FieldType.Kind() == reflect.Struct {
				ef.StructField.Index = append([]int{fieldStruct.Index[0]}, ef.StructField.Index...)
			} else {
				ef.StructField.Index = append([]int{-fieldStruct.Index[0] - 1}, ef.StructField.Index...)
			}

			if prefix, ok := field.TagSettings["EMBEDDEDPREFIX"]; ok {
				ef.DBName = prefix + ef.DBName
			}

			for k, v := range field.TagSettings {
				ef.TagSettings[k] = v
			}
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
				if fieldValue.IsNil() {
					fieldValue.Set(reflect.New(field.FieldType.Elem()))
				}
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
			for _, idx := range field.StructField.Index {
				if idx >= 0 {
					v = v.Field(idx)
				} else {
					v = v.Field(-idx - 1)
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

	recoverFunc := func(value reflect.Value, v interface{}, setter func(reflect.Value, interface{}) error) (err error) {
		reflectV := reflect.ValueOf(v)
		if reflectV.Type().ConvertibleTo(field.FieldType) {
			field.ReflectValueOf(value).Set(reflectV.Convert(field.FieldType))
		} else if valuer, ok := v.(driver.Valuer); ok {
			if v, err = valuer.Value(); err == nil {
				return setter(value, v)
			}
		} else if field.FieldType.Kind() == reflect.Ptr && reflectV.Type().ConvertibleTo(field.FieldType.Elem()) {
			field.ReflectValueOf(value).Elem().Set(reflectV.Convert(field.FieldType.Elem()))
		} else if reflectV.Kind() == reflect.Ptr {
			return field.Set(value, reflectV.Elem().Interface())
		} else {
			return fmt.Errorf("failed to set value %+v to field %v", v, field.Name)
		}
		return err
	}

	// Set
	switch field.FieldType.Kind() {
	case reflect.Bool:
		field.Set = func(value reflect.Value, v interface{}) error {
			switch data := v.(type) {
			case bool:
				field.ReflectValueOf(value).SetBool(data)
			case *bool:
				field.ReflectValueOf(value).SetBool(*data)
			default:
				return recoverFunc(value, v, field.Set)
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
				if field.AutoCreateTime == UnixNanosecond {
					field.ReflectValueOf(value).SetInt(data.UnixNano())
				} else {
					field.ReflectValueOf(value).SetInt(data.Unix())
				}
			case *time.Time:
				if data != nil {
					if field.AutoCreateTime == UnixNanosecond {
						field.ReflectValueOf(value).SetInt(data.UnixNano())
					} else {
						field.ReflectValueOf(value).SetInt(data.Unix())
					}
				} else {
					field.ReflectValueOf(value).SetInt(0)
				}
			default:
				return recoverFunc(value, v, field.Set)
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
			case string:
				if i, err := strconv.ParseUint(data, 0, 64); err == nil {
					field.ReflectValueOf(value).SetUint(i)
				} else {
					return err
				}
			default:
				return recoverFunc(value, v, field.Set)
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
				return recoverFunc(value, v, field.Set)
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
				field.ReflectValueOf(value).SetString(fmt.Sprint(data))
			case float64, float32:
				field.ReflectValueOf(value).SetString(fmt.Sprintf("%."+strconv.Itoa(field.Precision)+"f", data))
			default:
				return recoverFunc(value, v, field.Set)
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
					field.ReflectValueOf(value).Set(reflect.ValueOf(v).Elem())
				case string:
					if t, err := now.Parse(data); err == nil {
						field.ReflectValueOf(value).Set(reflect.ValueOf(t))
					} else {
						return fmt.Errorf("failed to set string %v to time.Time field %v, failed to parse it as time, got error %v", v, field.Name, err)
					}
				default:
					return recoverFunc(value, v, field.Set)
				}
				return nil
			}
		case *time.Time:
			field.Set = func(value reflect.Value, v interface{}) error {
				switch data := v.(type) {
				case time.Time:
					field.ReflectValueOf(value).Elem().Set(reflect.ValueOf(v))
				case *time.Time:
					field.ReflectValueOf(value).Set(reflect.ValueOf(v))
				case string:
					if t, err := now.Parse(data); err == nil {
						field.ReflectValueOf(value).Elem().Set(reflect.ValueOf(t))
					} else {
						return fmt.Errorf("failed to set string %v to time.Time field %v, failed to parse it as time, got error %v", v, field.Name, err)
					}
				default:
					return recoverFunc(value, v, field.Set)
				}
				return nil
			}
		default:
			if _, ok := fieldValue.Interface().(sql.Scanner); ok {
				// struct scanner
				field.Set = func(value reflect.Value, v interface{}) (err error) {
					reflectV := reflect.ValueOf(v)
					if reflectV.Type().ConvertibleTo(field.FieldType) {
						field.ReflectValueOf(value).Set(reflectV.Convert(field.FieldType))
					} else if valuer, ok := v.(driver.Valuer); ok {
						if v, err = valuer.Value(); err == nil {
							err = field.ReflectValueOf(value).Addr().Interface().(sql.Scanner).Scan(v)
						}
					} else {
						err = field.ReflectValueOf(value).Addr().Interface().(sql.Scanner).Scan(v)
					}
					return
				}
			} else if _, ok := fieldValue.Elem().Interface().(sql.Scanner); ok {
				// pointer scanner
				field.Set = func(value reflect.Value, v interface{}) (err error) {
					reflectV := reflect.ValueOf(v)
					if reflectV.Type().ConvertibleTo(field.FieldType) {
						field.ReflectValueOf(value).Set(reflectV.Convert(field.FieldType))
					} else if reflectV.Type().ConvertibleTo(field.FieldType.Elem()) {
						field.ReflectValueOf(value).Elem().Set(reflectV.Convert(field.FieldType.Elem()))
					} else if valuer, ok := v.(driver.Valuer); ok {
						if v, err = valuer.Value(); err == nil {
							err = field.ReflectValueOf(value).Interface().(sql.Scanner).Scan(v)
						}
					} else {
						err = field.ReflectValueOf(value).Interface().(sql.Scanner).Scan(v)
					}
					return
				}
			} else {
				field.Set = func(value reflect.Value, v interface{}) (err error) {
					return recoverFunc(value, v, field.Set)
				}
			}
		}
	}
}
