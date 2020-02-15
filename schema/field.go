package schema

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/jinzhu/now"
)

type DataType string

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
	Name            string
	DBName          string
	BindNames       []string
	DataType        DataType
	DBDataType      string
	PrimaryKey      bool
	AutoIncrement   bool
	Creatable       bool
	Updatable       bool
	HasDefaultValue bool
	DefaultValue    string
	NotNull         bool
	Unique          bool
	Comment         string
	Size            int
	Precision       int
	FieldType       reflect.Type
	StructField     reflect.StructField
	Tag             reflect.StructTag
	TagSettings     map[string]string
	Schema          *Schema
	EmbeddedSchema  *Schema
	ReflectValuer   func(reflect.Value) reflect.Value
	Valuer          func(reflect.Value) interface{}
	Setter          func(reflect.Value, interface{}) error
}

func (schema *Schema) ParseField(fieldStruct reflect.StructField) *Field {
	field := &Field{
		Name:        fieldStruct.Name,
		BindNames:   []string{fieldStruct.Name},
		FieldType:   fieldStruct.Type,
		StructField: fieldStruct,
		Creatable:   true,
		Updatable:   true,
		Tag:         fieldStruct.Tag,
		TagSettings: ParseTagSetting(fieldStruct.Tag),
		Schema:      schema,
	}

	for field.FieldType.Kind() == reflect.Ptr {
		field.FieldType = field.FieldType.Elem()
	}

	fieldValue := reflect.New(field.FieldType)

	// if field is valuer, used its value or first fields as data type
	if valuer, isValuer := fieldValue.Interface().(driver.Valuer); isValuer {
		var overrideFieldValue bool
		if v, err := valuer.Value(); v != nil && err == nil {
			overrideFieldValue = true
			fieldValue = reflect.ValueOf(v)
		}

		if field.FieldType.Kind() == reflect.Struct {
			for i := 0; i < field.FieldType.NumField(); i++ {
				if !overrideFieldValue {
					newFieldType := field.FieldType.Field(i).Type
					for newFieldType.Kind() == reflect.Ptr {
						newFieldType = newFieldType.Elem()
					}

					fieldValue = reflect.New(newFieldType)
					overrideFieldValue = true
				}

				// copy tag settings from valuer
				for key, value := range ParseTagSetting(field.FieldType.Field(i).Tag) {
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
		field.Size, _ = strconv.Atoi(num)
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
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		field.DataType = Int
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		field.DataType = Uint
	case reflect.Float32, reflect.Float64:
		field.DataType = Float
	case reflect.String:
		field.DataType = String
	case reflect.Struct:
		if _, ok := fieldValue.Interface().(*time.Time); ok {
			field.DataType = Time
		}
	case reflect.Array, reflect.Slice:
		if fieldValue.Type().Elem() == reflect.TypeOf(uint8(0)) {
			field.DataType = Bytes
		}
	}

	if field.Size == 0 {
		switch fieldValue.Kind() {
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
				ef.StructField.Index = append([]int{-fieldStruct.Index[0]}, ef.StructField.Index...)
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

// ValueOf field value of
func (field *Field) ValueOf(value reflect.Value) interface{} {
	if field != nil {
		return field.Valuer(value)
	}
	return nil
}

func (field *Field) Set(value reflect.Value, v interface{}) error {
	if field != nil {
		return field.Setter(value, v)
	}

	return fmt.Errorf("failed to set field value: %v", field.Name)
}

// create valuer, setter when parse struct
func (field *Field) setupValuerAndSetter() {
	// Valuer
	switch {
	case len(field.StructField.Index) == 1:
		field.Valuer = func(value reflect.Value) interface{} {
			return value.Field(field.StructField.Index[0]).Interface()
		}
	case len(field.StructField.Index) == 2 && field.StructField.Index[0] >= 0:
		field.Valuer = func(value reflect.Value) interface{} {
			return value.Field(field.StructField.Index[0]).Field(field.StructField.Index[1]).Interface()
		}
	default:
		field.Valuer = func(value reflect.Value) interface{} {
			v := value.Field(field.StructField.Index[0])
			for _, idx := range field.StructField.Index[1:] {
				if v.Kind() == reflect.Ptr {
					if v.Type().Elem().Kind() == reflect.Struct {
						if !v.IsNil() {
							v = v.Elem().Field(-idx)
							continue
						}
					}
					return nil
				} else {
					v = v.Field(idx)
				}
			}
			return v.Interface()
		}
	}

	// ReflectValuer
	switch {
	case len(field.StructField.Index) == 1:
		if field.FieldType.Kind() == reflect.Ptr {
			field.ReflectValuer = func(value reflect.Value) reflect.Value {
				fieldValue := value.Field(field.StructField.Index[0])
				if fieldValue.IsNil() {
					fieldValue.Set(reflect.New(field.FieldType.Elem()))
				}
				return fieldValue
			}
		} else {
			field.ReflectValuer = func(value reflect.Value) reflect.Value {
				return value.Field(field.StructField.Index[0])
			}
		}
	case len(field.StructField.Index) == 2 && field.StructField.Index[0] >= 0 && field.FieldType.Kind() != reflect.Ptr:
		field.Valuer = func(value reflect.Value) interface{} {
			return value.Field(field.StructField.Index[0]).Field(field.StructField.Index[1]).Interface()
		}
	default:
		field.ReflectValuer = func(value reflect.Value) reflect.Value {
			v := value.Field(field.StructField.Index[0])
			for _, idx := range field.StructField.Index[1:] {
				if v.Kind() == reflect.Ptr {
					if v.Type().Elem().Kind() == reflect.Struct {
						if v.IsNil() {
							v.Set(reflect.New(v.Type().Elem()))
						}

						if idx >= 0 {
							v = v.Elem().Field(idx)
						} else {
							v = v.Elem().Field(-idx)
						}
					}
				} else {
					v = v.Field(idx)
				}
			}
			return v
		}
	}

	// Setter
	switch field.FieldType.Kind() {
	case reflect.Bool:
		field.Setter = func(value reflect.Value, v interface{}) error {
			switch data := v.(type) {
			case bool:
				field.ReflectValuer(value).SetBool(data)
			case *bool:
				field.ReflectValuer(value).SetBool(*data)
			default:
				reflectV := reflect.ValueOf(v)
				if reflectV.Type().ConvertibleTo(field.FieldType) {
					field.ReflectValuer(value).Set(reflectV.Convert(field.FieldType))
				} else {
					field.ReflectValuer(value).SetBool(!reflect.ValueOf(v).IsZero())
				}
			}
			return nil
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		field.Setter = func(value reflect.Value, v interface{}) error {
			switch data := v.(type) {
			case int64:
				field.ReflectValuer(value).SetInt(data)
			case int:
				field.ReflectValuer(value).SetInt(int64(data))
			case int8:
				field.ReflectValuer(value).SetInt(int64(data))
			case int16:
				field.ReflectValuer(value).SetInt(int64(data))
			case int32:
				field.ReflectValuer(value).SetInt(int64(data))
			case uint:
				field.ReflectValuer(value).SetInt(int64(data))
			case uint8:
				field.ReflectValuer(value).SetInt(int64(data))
			case uint16:
				field.ReflectValuer(value).SetInt(int64(data))
			case uint32:
				field.ReflectValuer(value).SetInt(int64(data))
			case uint64:
				field.ReflectValuer(value).SetInt(int64(data))
			case float32:
				field.ReflectValuer(value).SetInt(int64(data))
			case float64:
				field.ReflectValuer(value).SetInt(int64(data))
			case []byte:
				return field.Setter(value, string(data))
			case string:
				if i, err := strconv.ParseInt(data, 0, 64); err == nil {
					field.ReflectValuer(value).SetInt(i)
				} else {
					return err
				}
			default:
				reflectV := reflect.ValueOf(v)
				if reflectV.Type().ConvertibleTo(field.FieldType) {
					field.ReflectValuer(value).Set(reflectV.Convert(field.FieldType))
				} else if reflectV.Kind() == reflect.Ptr {
					return field.Setter(value, reflectV.Elem().Interface())
				} else {
					return fmt.Errorf("failed to set value %+v to field %v", v, field.Name)
				}
			}
			return nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		field.Setter = func(value reflect.Value, v interface{}) error {
			switch data := v.(type) {
			case uint64:
				field.ReflectValuer(value).SetUint(data)
			case uint:
				field.ReflectValuer(value).SetUint(uint64(data))
			case uint8:
				field.ReflectValuer(value).SetUint(uint64(data))
			case uint16:
				field.ReflectValuer(value).SetUint(uint64(data))
			case uint32:
				field.ReflectValuer(value).SetUint(uint64(data))
			case int64:
				field.ReflectValuer(value).SetUint(uint64(data))
			case int:
				field.ReflectValuer(value).SetUint(uint64(data))
			case int8:
				field.ReflectValuer(value).SetUint(uint64(data))
			case int16:
				field.ReflectValuer(value).SetUint(uint64(data))
			case int32:
				field.ReflectValuer(value).SetUint(uint64(data))
			case float32:
				field.ReflectValuer(value).SetUint(uint64(data))
			case float64:
				field.ReflectValuer(value).SetUint(uint64(data))
			case []byte:
				return field.Setter(value, string(data))
			case string:
				if i, err := strconv.ParseUint(data, 0, 64); err == nil {
					field.ReflectValuer(value).SetUint(i)
				} else {
					return err
				}
			default:
				reflectV := reflect.ValueOf(v)
				if reflectV.Type().ConvertibleTo(field.FieldType) {
					field.ReflectValuer(value).Set(reflectV.Convert(field.FieldType))
				} else if reflectV.Kind() == reflect.Ptr {
					return field.Setter(value, reflectV.Elem().Interface())
				} else {
					return fmt.Errorf("failed to set value %+v to field %v", v, field.Name)
				}
			}
			return nil
		}
	case reflect.Float32, reflect.Float64:
		field.Setter = func(value reflect.Value, v interface{}) error {
			switch data := v.(type) {
			case float64:
				field.ReflectValuer(value).SetFloat(data)
			case float32:
				field.ReflectValuer(value).SetFloat(float64(data))
			case int64:
				field.ReflectValuer(value).SetFloat(float64(data))
			case int:
				field.ReflectValuer(value).SetFloat(float64(data))
			case int8:
				field.ReflectValuer(value).SetFloat(float64(data))
			case int16:
				field.ReflectValuer(value).SetFloat(float64(data))
			case int32:
				field.ReflectValuer(value).SetFloat(float64(data))
			case uint:
				field.ReflectValuer(value).SetFloat(float64(data))
			case uint8:
				field.ReflectValuer(value).SetFloat(float64(data))
			case uint16:
				field.ReflectValuer(value).SetFloat(float64(data))
			case uint32:
				field.ReflectValuer(value).SetFloat(float64(data))
			case uint64:
				field.ReflectValuer(value).SetFloat(float64(data))
			case []byte:
				return field.Setter(value, string(data))
			case string:
				if i, err := strconv.ParseFloat(data, 64); err == nil {
					field.ReflectValuer(value).SetFloat(i)
				} else {
					return err
				}
			default:
				reflectV := reflect.ValueOf(v)
				if reflectV.Type().ConvertibleTo(field.FieldType) {
					field.ReflectValuer(value).Set(reflectV.Convert(field.FieldType))
				} else if reflectV.Kind() == reflect.Ptr {
					return field.Setter(value, reflectV.Elem().Interface())
				} else {
					return fmt.Errorf("failed to set value %+v to field %v", v, field.Name)
				}
			}
			return nil
		}
	case reflect.String:
		field.Setter = func(value reflect.Value, v interface{}) error {
			switch data := v.(type) {
			case string:
				field.ReflectValuer(value).SetString(data)
			case []byte:
				field.ReflectValuer(value).SetString(string(data))
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				field.ReflectValuer(value).SetString(fmt.Sprint(data))
			case float64, float32:
				field.ReflectValuer(value).SetString(fmt.Sprintf("%."+strconv.Itoa(field.Precision)+"f", data))
			default:
				reflectV := reflect.ValueOf(v)
				if reflectV.Type().ConvertibleTo(field.FieldType) {
					field.ReflectValuer(value).Set(reflectV.Convert(field.FieldType))
				} else if reflectV.Kind() == reflect.Ptr {
					return field.Setter(value, reflectV.Elem().Interface())
				} else {
					return fmt.Errorf("failed to set value %+v to field %v", v, field.Name)
				}
			}
			return nil
		}
	default:
		fieldValue := reflect.New(field.FieldType)
		switch fieldValue.Interface().(type) {
		case time.Time:
			field.Setter = func(value reflect.Value, v interface{}) error {
				switch data := v.(type) {
				case time.Time:
					field.ReflectValuer(value).Set(reflect.ValueOf(v))
				case *time.Time:
					field.ReflectValuer(value).Set(reflect.ValueOf(v).Elem())
				case string:
					if t, err := now.Parse(data); err == nil {
						field.ReflectValuer(value).Set(reflect.ValueOf(t))
					} else {
						return fmt.Errorf("failed to set string %v to time.Time field %v, failed to parse it as time, got error %v", v, field.Name, err)
					}
				default:
					return fmt.Errorf("failed to set value %+v to time.Time field %v", v, field.Name)
				}
				return nil
			}
		case *time.Time:
			field.Setter = func(value reflect.Value, v interface{}) error {
				switch data := v.(type) {
				case time.Time:
					field.ReflectValuer(value).Elem().Set(reflect.ValueOf(v))
				case *time.Time:
					field.ReflectValuer(value).Set(reflect.ValueOf(v))
				case string:
					if t, err := now.Parse(data); err == nil {
						field.ReflectValuer(value).Elem().Set(reflect.ValueOf(t))
					} else {
						return fmt.Errorf("failed to set string %v to time.Time field %v, failed to parse it as time, got error %v", v, field.Name, err)
					}
				default:
					return fmt.Errorf("failed to set value %+v to time.Time field %v", v, field.Name)
				}
				return nil
			}
		default:
			if fieldValue.CanAddr() {
				if _, ok := fieldValue.Addr().Interface().(sql.Scanner); ok {
					field.Setter = func(value reflect.Value, v interface{}) (err error) {
						if valuer, ok := v.(driver.Valuer); ok {
							if v, err = valuer.Value(); err == nil {
								err = field.ReflectValuer(value).Addr().Interface().(sql.Scanner).Scan(v)
							}
						} else {
							err = field.ReflectValuer(value).Addr().Interface().(sql.Scanner).Scan(v)
						}
						return
					}
					return
				}
			}

			field.Setter = func(value reflect.Value, v interface{}) (err error) {
				reflectV := reflect.ValueOf(v)
				if reflectV.Type().ConvertibleTo(field.FieldType) {
					field.ReflectValuer(value).Set(reflectV.Convert(field.FieldType))
				} else {
					return fmt.Errorf("failed to set value %+v to field %v", v, field.Name)
				}
				return nil
			}
		}
	}
}
