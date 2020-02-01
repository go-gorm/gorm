package schema

import (
	"database/sql/driver"
	"reflect"
	"strconv"
	"sync"
	"time"
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
		field.EmbeddedSchema, schema.err = Parse(fieldValue.Interface(), &sync.Map{}, schema.namer)
		for _, ef := range field.EmbeddedSchema.Fields {
			ef.BindNames = append([]string{fieldStruct.Name}, ef.BindNames...)

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
