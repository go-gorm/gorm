package schema

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jinzhu/now"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/utils"
)

// special types' reflect type
var (
	// TimeReflectType is the reflect.Type of time.Time, used when classifying
	// time-compatible model fields. Treat this package variable as read-only.
	TimeReflectType = reflect.TypeOf(time.Time{})
	// TimePtrReflectType is the reflect.Type of *time.Time, used to recognize
	// pointer-compatible time fields. Treat it as read-only.
	TimePtrReflectType = reflect.TypeOf(&time.Time{})
	// ByteReflectType is the reflect.Type of uint8, used to classify byte slices and
	// arrays. Treat it as read-only.
	ByteReflectType = reflect.TypeOf(uint8(0))
)

type (
	// DataType names a logical GORM column type or an explicit database type from a model
	// tag. Dialectors map logical types to database-specific SQL.
	DataType string
	// TimeType selects the representation used for automatic timestamps. Zero disables
	// automatic timestamp handling; UnixTime stores time.Time and the other values select
	// integer epoch units.
	TimeType int64
)

// GORM time types
const (
	// UnixTime marks automatic timestamps stored as time.Time values.
	UnixTime TimeType = 1
	// UnixSecond marks automatic timestamps stored as seconds since the Unix epoch.
	UnixSecond TimeType = 2
	// UnixMillisecond marks automatic timestamps stored as milliseconds since the Unix
	// epoch.
	UnixMillisecond TimeType = 3
	// UnixNanosecond marks automatic timestamps stored as nanoseconds since the Unix epoch.
	UnixNanosecond TimeType = 4
)

// GORM fields types
const (
	// Bool is the logical GORM type for boolean columns. The dialector selects the concrete
	// database type.
	Bool DataType = "bool"
	// Int is the logical GORM type for signed integer columns. The dialector selects the
	// concrete database type.
	Int DataType = "int"
	// Uint is the logical GORM type for unsigned integer columns. The dialector selects the
	// concrete database type.
	Uint DataType = "uint"
	// Float is the logical GORM type for floating-point columns. The dialector selects the
	// concrete database type.
	Float DataType = "float"
	// String is the logical GORM type for text columns. The dialector selects the concrete
	// database type.
	String DataType = "string"
	// Time is the logical GORM type for date/time columns. The dialector selects the
	// concrete database type.
	Time DataType = "time"
	// Bytes is the logical GORM type for binary columns. The dialector selects the concrete
	// database type.
	Bytes DataType = "bytes"
)

// DefaultAutoIncrementIncrement is the default spacing between generated keys when
// create callbacks backfill batch IDs. The autoIncrementIncrement tag overrides it; it
// does not configure the database sequence.
const DefaultAutoIncrementIncrement int64 = 1

// Field describes a model field, including its column mapping, permissions, defaults,
// embedding path, and compiled accessors. Not every Field is a column: associations and
// ignored fields can have an empty DataType or DBName.
//
// Fields are shared through cached schemas. Use the accessors with a value of the
// owning model; setters and embedding allocation require addressable values. These
// accessors do not enforce CRUD permissions by themselves: statement and callback code
// applies Creatable, Updatable, and Readable.
type Field struct {
	// Name is the Go struct field name, without its embedding path.
	Name string
	// DBName is the database column name from the column tag or naming strategy; it can be
	// empty for associations or ignored fields.
	DBName string
	// BindNames is the Go field path from the root model, including anonymous embedding.
	// BindName joins it with dots.
	BindNames []string
	// EmbeddedBindNames is the association path used for explicitly embedded fields;
	// ordinary anonymous embedding can be omitted from this path.
	EmbeddedBindNames []string
	// DataType is the effective logical or explicit SQL type after applying the type tag.
	// Dialectors interpret it when generating column definitions.
	DataType DataType
	// GORMDataType preserves the inferred or GormDataType-provided logical type before a
	// type tag overrides DataType. Plugins can use it to identify the underlying field
	// category.
	GORMDataType DataType
	// PrimaryKey indicates membership in the model primary key, whether declared by tag or
	// inferred from ID.
	PrimaryKey bool
	// AutoIncrement indicates a database-generated incrementing key. Integer prioritized
	// primary keys enable it by convention unless explicitly configured otherwise.
	AutoIncrement bool
	// AutoIncrementIncrement is the step used to calculate generated IDs for a batch from
	// LastInsertId; it defaults to DefaultAutoIncrementIncrement.
	AutoIncrementIncrement int64
	// Creatable permits this field in model-based inserts, subject to Select/Omit and
	// default-value handling.
	Creatable bool
	// Updatable permits this field in model-based updates, subject to Select/Omit and
	// zero-value handling.
	Updatable bool
	// Readable permits database results to be assigned to this model field during GORM
	// scanning.
	Readable bool
	// AutoCreateTime selects automatic creation-time handling and its storage unit; zero
	// disables it.
	AutoCreateTime TimeType
	// AutoUpdateTime selects automatic update-time handling and its storage unit; zero
	// disables it. Callbacks decide whether an operation updates timestamps.
	AutoUpdateTime TimeType
	// HasDefaultValue records a declared or inferred default, including auto-increment
	// keys. A zero Go value may therefore be omitted or replaced during creation.
	HasDefaultValue bool
	// DefaultValue holds the trimmed default tag expression, with string-literal quotes
	// removed when parsed. Database expressions are trusted model metadata, not bound user
	// values.
	DefaultValue string
	// DefaultValueInterface holds a successfully parsed Go literal default. Nil also
	// represents defaults left to the database, such as functions, NULL, and generated
	// keys; consult HasDefaultValue.
	DefaultValueInterface interface{}
	// NotNull requests a NOT NULL column constraint during migration; it does not validate
	// Go values before writing.
	NotNull bool
	// Unique requests a single-column unique constraint. Unique indexes are tracked
	// separately by ParseIndexes and UniqueIndex.
	Unique bool
	// Comment is the column comment requested by the comment tag; support depends on the
	// dialector.
	Comment string
	// Size is the declared column length or inferred numeric bit width. Its SQL meaning
	// depends on the dialector and DataType.
	Size int
	// Precision is the precision requested by the precision tag for dialector-specific
	// column definitions.
	Precision int
	// Scale is the scale requested by the scale tag, commonly the fractional digit count
	// for decimal columns.
	Scale int
	// IgnoreMigration excludes this field from migration handling without necessarily
	// disabling reads or writes.
	IgnoreMigration bool
	// FieldType is the declared Go type, preserving pointer wrappers.
	FieldType reflect.Type
	// IndirectFieldType is FieldType with pointer wrappers removed.
	IndirectFieldType reflect.Type
	// StructField contains the reflection metadata. Its Index path is rewritten for
	// embedded fields; a negative component encodes an embedded pointer as -index-1.
	StructField reflect.StructField
	// Tag is the original Go struct tag for this field.
	Tag reflect.StructTag
	// TagSettings contains parsed GORM tags with uppercase keys, including inherited
	// embedded-field settings.
	TagSettings map[string]string
	// Schema is the root model schema that owns this field after embedding is flattened.
	Schema *Schema
	// EmbeddedSchema describes the nested struct when this field is an embedding container;
	// it is nil for ordinary columns and associations.
	EmbeddedSchema *Schema
	// OwnerSchema is the original embedded schema for a flattened field, or nil when no
	// embedded owner is recorded.
	OwnerSchema *Schema
	// ReflectValueOf locates the field in a model value and allocates nil embedded pointers
	// on the path. The model must have the expected shape and be addressable when
	// allocation or mutation is needed.
	ReflectValueOf func(context.Context, reflect.Value) reflect.Value
	// ValueOf returns the field value and its Go zero-value status. A nil embedded pointer
	// yields (nil, true). Serialized fields return a driver.Valuer wrapper while preserving
	// the underlying zero flag.
	ValueOf func(context.Context, reflect.Value) (value interface{}, zero bool)
	// Set assigns or converts a value into an addressable model field, using type-specific
	// conversions, sql.Scanner, or the configured serializer. Conversion semantics are
	// type-dependent; inspect the returned error rather than assuming validation or
	// lossless numeric conversion.
	Set func(context.Context, reflect.Value, interface{}) error
	// Serializer is the codec selected by the field type or serializer/json tag. It is
	// captured while parsing the schema.
	Serializer SerializerInterface
	// NewValuePool supplies temporary scan destinations. Callers must return each borrowed
	// object after assigning it with Set, and must not retain it afterward.
	NewValuePool FieldNewValuePool

	// UniqueIndex records the name of a single-column UNIQUE index during ParseIndexes.
	// Migrators use it to distinguish tagged indexes from Unique constraints on databases
	// that report both as column uniqueness; composite unique indexes do not populate it.
	UniqueIndex string
}

// BindName returns the dotted Go embedding path, such as Address.City, used by
// Schema.FieldsByBindName.
func (field *Field) BindName() string {
	return strings.Join(field.BindNames, ".")
}

// ParseField interprets one reflected struct field and its gorm tag, including logical
// type inference, driver.Valuer and serializer support, defaults, automatic timestamps,
// permissions, and embedding. It records failures on the schema and returns the field
// even when parsing fails.
//
// This is one stage of Parse: column naming, primary-key prioritization, relationship
// resolution, and accessors are completed by the full schema parser. Calling ParseField
// alone does not produce a fully usable independent field. Anonymous structs or fields
// tagged embedded are flattened by the caller; embeddedPrefix changes their column
// names.
func (schema *Schema) ParseField(fieldStruct reflect.StructField) *Field {
	var (
		err        error
		tagSetting = ParseTagSetting(fieldStruct.Tag.Get("gorm"), ";")
	)

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
		HasDefaultValue:        utils.CheckTruth(tagSetting["AUTOINCREMENT"]),
		NotNull:                utils.CheckTruth(tagSetting["NOT NULL"], tagSetting["NOTNULL"]),
		Unique:                 utils.CheckTruth(tagSetting["UNIQUE"]),
		Comment:                tagSetting["COMMENT"],
		AutoIncrementIncrement: DefaultAutoIncrementIncrement,
	}

	for field.IndirectFieldType.Kind() == reflect.Ptr {
		field.IndirectFieldType = field.IndirectFieldType.Elem()
	}

	fieldValue := reflect.New(field.IndirectFieldType)
	// if field is valuer, used its value or first field as data type
	valuer, isValuer := fieldValue.Interface().(driver.Valuer)
	if isValuer {
		if _, ok := fieldValue.Interface().(GormDataTypeInterface); !ok {
			if v, err := valuer.Value(); reflect.ValueOf(v).IsValid() && err == nil {
				fieldValue = reflect.ValueOf(v)
			}

			// Use the field struct's first field type as data type, e.g: use `string` for sql.NullString
			var getRealFieldValue func(reflect.Value)
			getRealFieldValue = func(v reflect.Value) {
				var (
					rv     = reflect.Indirect(v)
					rvType = rv.Type()
				)

				if rv.Kind() == reflect.Struct && !rvType.ConvertibleTo(TimeReflectType) {
					for i := 0; i < rvType.NumField(); i++ {
						for key, value := range ParseTagSetting(rvType.Field(i).Tag.Get("gorm"), ";") {
							if _, ok := field.TagSettings[key]; !ok {
								field.TagSettings[key] = value
							}
						}
					}

					for i := 0; i < rvType.NumField(); i++ {
						newFieldType := rvType.Field(i).Type
						for newFieldType.Kind() == reflect.Ptr {
							newFieldType = newFieldType.Elem()
						}

						fieldValue = reflect.New(newFieldType)
						if rvType != reflect.Indirect(fieldValue).Type() {
							getRealFieldValue(fieldValue)
						}

						if fieldValue.IsValid() {
							return
						}
					}
				}
			}

			getRealFieldValue(fieldValue)
		}
	}

	if v, isSerializer := fieldValue.Interface().(SerializerInterface); isSerializer {
		field.DataType = String
		field.Serializer = v
	} else {
		serializerName := field.TagSettings["JSON"]
		if serializerName == "" {
			serializerName = field.TagSettings["SERIALIZER"]
		}
		if serializerName != "" {
			if serializer, ok := GetSerializer(serializerName); ok {
				// Set default data type to string for serializer
				field.DataType = String
				field.Serializer = serializer
			} else {
				schema.err = fmt.Errorf("invalid serializer type %v", serializerName)
			}
		}
	}

	if num, ok := field.TagSettings["AUTOINCREMENTINCREMENT"]; ok {
		field.AutoIncrementIncrement, _ = strconv.ParseInt(num, 10, 64)
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

	// default value is function or null or blank (primary keys)
	field.DefaultValue = strings.TrimSpace(field.DefaultValue)
	skipParseDefaultValue := strings.Contains(field.DefaultValue, "(") &&
		strings.Contains(field.DefaultValue, ")") || strings.ToLower(field.DefaultValue) == "null" || field.DefaultValue == ""
	switch reflect.Indirect(fieldValue).Kind() {
	case reflect.Bool:
		field.DataType = Bool
		if field.HasDefaultValue && !skipParseDefaultValue {
			if field.DefaultValueInterface, err = strconv.ParseBool(field.DefaultValue); err != nil {
				schema.err = fmt.Errorf("failed to parse %s as default value for bool, got error: %v", field.DefaultValue, err)
			}
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		field.DataType = Int
		if field.HasDefaultValue && !skipParseDefaultValue {
			if field.DefaultValueInterface, err = strconv.ParseInt(field.DefaultValue, 0, 64); err != nil {
				schema.err = fmt.Errorf("failed to parse %s as default value for int, got error: %v", field.DefaultValue, err)
			}
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		field.DataType = Uint
		if field.HasDefaultValue && !skipParseDefaultValue {
			if field.DefaultValueInterface, err = strconv.ParseUint(field.DefaultValue, 0, 64); err != nil {
				schema.err = fmt.Errorf("failed to parse %s as default value for uint, got error: %v", field.DefaultValue, err)
			}
		}
	case reflect.Float32, reflect.Float64:
		field.DataType = Float
		if field.HasDefaultValue && !skipParseDefaultValue {
			if field.DefaultValueInterface, err = strconv.ParseFloat(field.DefaultValue, 64); err != nil {
				schema.err = fmt.Errorf("failed to parse %s as default value for float, got error: %v", field.DefaultValue, err)
			}
		}
	case reflect.String:
		field.DataType = String
		if field.HasDefaultValue && !skipParseDefaultValue {
			field.DefaultValue = strings.Trim(field.DefaultValue, "'")
			field.DefaultValue = strings.Trim(field.DefaultValue, `"`)
			field.DefaultValueInterface = field.DefaultValue
		}
	case reflect.Struct:
		if _, ok := fieldValue.Interface().(*time.Time); ok {
			field.DataType = Time
		} else if fieldValue.Type().ConvertibleTo(TimeReflectType) {
			field.DataType = Time
		} else if fieldValue.Type().ConvertibleTo(TimePtrReflectType) {
			field.DataType = Time
		}
		if field.HasDefaultValue && !skipParseDefaultValue && field.DataType == Time {
			if t, err := now.Parse(field.DefaultValue); err == nil {
				field.DefaultValueInterface = t
			}
		}
	case reflect.Array, reflect.Slice:
		if reflect.Indirect(fieldValue).Type().Elem() == ByteReflectType && field.DataType == "" {
			field.DataType = Bytes
		}
	}

	if dataTyper, ok := fieldValue.Interface().(GormDataTypeInterface); ok {
		field.DataType = DataType(dataTyper.GormDataType())
	}

	if v, ok := field.TagSettings["AUTOCREATETIME"]; (ok && utils.CheckTruth(v)) || (!ok && field.Name == "CreatedAt" && (field.DataType == Time || field.DataType == Int || field.DataType == Uint)) {
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

	if v, ok := field.TagSettings["AUTOUPDATETIME"]; (ok && utils.CheckTruth(v)) || (!ok && field.Name == "UpdatedAt" && (field.DataType == Time || field.DataType == Int || field.DataType == Uint)) {
		if field.DataType == Time {
			field.AutoUpdateTime = UnixTime
		} else if strings.ToUpper(v) == "NANO" {
			field.AutoUpdateTime = UnixNanosecond
		} else if strings.ToUpper(v) == "MILLI" {
			field.AutoUpdateTime = UnixMillisecond
		} else {
			field.AutoUpdateTime = UnixSecond
		}
	}

	if field.GORMDataType == "" {
		field.GORMDataType = field.DataType
	}

	if val, ok := field.TagSettings["TYPE"]; ok {
		lowerVal := DataType(strings.ToLower(val))
		switch lowerVal {
		case Bool, Int, Uint, Float, String, Time, Bytes:
			field.DataType = lowerVal
		default:
			field.DataType = DataType(val)
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

	// setup permission
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

	// Normal anonymous field or having `EMBEDDED` tag
	if _, ok := field.TagSettings["EMBEDDED"]; ok || (field.GORMDataType != Time && field.GORMDataType != Bytes && !isValuer &&
		fieldStruct.Anonymous && (field.Creatable || field.Updatable || field.Readable)) {
		kind := reflect.Indirect(fieldValue).Kind()
		switch kind {
		case reflect.Struct:
			var err error
			field.Creatable = false
			field.Updatable = false
			field.Readable = false

			cacheStore := &sync.Map{}
			cacheStore.Store(embeddedCacheKey, true)
			if field.EmbeddedSchema, err = getOrParse(fieldValue.Interface(), cacheStore, embeddedNamer{Table: schema.Table, Namer: schema.namer}); err != nil {
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

	return field
}

// setupValuerAndSetter compiles field-path readers, allocating reflection access,
// type-specific assignment, and serializer wrappers for a parsed field. It initializes
// scan pooling first and expects the embedding index path to have been finalized.
func (field *Field) setupValuerAndSetter(modelType reflect.Type) {
	// Setup NewValuePool
	field.setupNewValuePool()

	// ValueOf returns field's value and if it is zero
	fieldIndex := field.StructField.Index[0]
	switch {
	case len(field.StructField.Index) == 1 && fieldIndex >= 0:
		field.ValueOf = func(ctx context.Context, v reflect.Value) (interface{}, bool) {
			v = reflect.Indirect(v)
			fieldValue := v.Field(fieldIndex)
			return fieldValue.Interface(), fieldValue.IsZero()
		}
	default:
		field.ValueOf = func(ctx context.Context, v reflect.Value) (interface{}, bool) {
			v = reflect.Indirect(v)
			for _, fieldIdx := range field.StructField.Index {
				if fieldIdx >= 0 {
					v = v.Field(fieldIdx)
				} else {
					v = v.Field(-fieldIdx - 1)

					if !v.IsNil() {
						v = v.Elem()
					} else {
						return nil, true
					}
				}
			}

			fv, zero := v.Interface(), v.IsZero()
			return fv, zero
		}
	}

	if field.Serializer != nil {
		oldValuerOf := field.ValueOf
		field.ValueOf = func(ctx context.Context, v reflect.Value) (interface{}, bool) {
			value, zero := oldValuerOf(ctx, v)

			s, ok := value.(SerializerValuerInterface)
			if !ok {
				s = field.Serializer
			}

			return &serializer{
				Field:           field,
				SerializeValuer: s,
				Destination:     v,
				Context:         ctx,
				fieldValue:      value,
			}, zero
		}
	}

	// ReflectValueOf returns field's reflect value
	switch {
	case len(field.StructField.Index) == 1 && fieldIndex >= 0:
		field.ReflectValueOf = func(ctx context.Context, v reflect.Value) reflect.Value {
			v = reflect.Indirect(v)
			return v.Field(fieldIndex)
		}
	default:
		field.ReflectValueOf = func(ctx context.Context, v reflect.Value) reflect.Value {
			v = reflect.Indirect(v)
			for idx, fieldIdx := range field.StructField.Index {
				if fieldIdx >= 0 {
					v = v.Field(fieldIdx)
				} else {
					v = v.Field(-fieldIdx - 1)

					if v.IsNil() {
						v.Set(reflect.New(v.Type().Elem()))
					}

					if idx < len(field.StructField.Index)-1 {
						v = v.Elem()
					}
				}
			}
			return v
		}
	}

	fallbackSetter := func(ctx context.Context, value reflect.Value, v interface{}, setter func(context.Context, reflect.Value, interface{}) error) (err error) {
		if v == nil {
			field.ReflectValueOf(ctx, value).Set(reflect.New(field.FieldType).Elem())
		} else {
			reflectV := reflect.ValueOf(v)
			// Optimal value type acquisition for v
			reflectValType := reflectV.Type()

			if reflectValType.AssignableTo(field.FieldType) {
				if reflectV.Kind() == reflect.Ptr && reflectV.Elem().Kind() == reflect.Ptr {
					reflectV = reflect.Indirect(reflectV)
				}
				field.ReflectValueOf(ctx, value).Set(reflectV)
				return
			} else if reflectValType.ConvertibleTo(field.FieldType) {
				field.ReflectValueOf(ctx, value).Set(reflectV.Convert(field.FieldType))
				return
			} else if field.FieldType.Kind() == reflect.Ptr {
				fieldValue := field.ReflectValueOf(ctx, value)
				fieldType := field.FieldType.Elem()

				if reflectValType.AssignableTo(fieldType) {
					if !fieldValue.IsValid() {
						fieldValue = reflect.New(fieldType)
					} else if fieldValue.IsNil() {
						fieldValue.Set(reflect.New(fieldType))
					}
					fieldValue.Elem().Set(reflectV)
					return
				} else if reflectValType.ConvertibleTo(fieldType) {
					if fieldValue.IsNil() {
						fieldValue.Set(reflect.New(fieldType))
					}

					fieldValue.Elem().Set(reflectV.Convert(fieldType))
					return
				}
			}

			if reflectV.Kind() == reflect.Ptr {
				if reflectV.IsNil() {
					field.ReflectValueOf(ctx, value).Set(reflect.New(field.FieldType).Elem())
				} else if reflectV.Type().Elem().AssignableTo(field.FieldType) {
					field.ReflectValueOf(ctx, value).Set(reflectV.Elem())
					return
				} else {
					err = setter(ctx, value, reflectV.Elem().Interface())
				}
			} else if valuer, ok := v.(driver.Valuer); ok {
				if v, err = valuer.Value(); err == nil {
					err = setter(ctx, value, v)
				}
			} else if _, ok := v.(clause.Expr); !ok {
				return fmt.Errorf("failed to set value %#v to field %s", v, field.Name)
			}
		}

		return
	}

	// Set
	switch field.FieldType.Kind() {
	case reflect.Bool:
		field.Set = func(ctx context.Context, value reflect.Value, v interface{}) error {
			switch data := v.(type) {
			case **bool:
				if data != nil && *data != nil {
					field.ReflectValueOf(ctx, value).SetBool(**data)
				}
			case bool:
				field.ReflectValueOf(ctx, value).SetBool(data)
			case int64:
				field.ReflectValueOf(ctx, value).SetBool(data > 0)
			case string:
				b, _ := strconv.ParseBool(data)
				field.ReflectValueOf(ctx, value).SetBool(b)
			default:
				return fallbackSetter(ctx, value, v, field.Set)
			}
			return nil
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		field.Set = func(ctx context.Context, value reflect.Value, v interface{}) (err error) {
			switch data := v.(type) {
			case **int64:
				if data != nil && *data != nil {
					field.ReflectValueOf(ctx, value).SetInt(**data)
				}
			case **int:
				if data != nil && *data != nil {
					field.ReflectValueOf(ctx, value).SetInt(int64(**data))
				}
			case **int8:
				if data != nil && *data != nil {
					field.ReflectValueOf(ctx, value).SetInt(int64(**data))
				}
			case **int16:
				if data != nil && *data != nil {
					field.ReflectValueOf(ctx, value).SetInt(int64(**data))
				}
			case **int32:
				if data != nil && *data != nil {
					field.ReflectValueOf(ctx, value).SetInt(int64(**data))
				}
			case int64:
				field.ReflectValueOf(ctx, value).SetInt(data)
			case int:
				field.ReflectValueOf(ctx, value).SetInt(int64(data))
			case int8:
				field.ReflectValueOf(ctx, value).SetInt(int64(data))
			case int16:
				field.ReflectValueOf(ctx, value).SetInt(int64(data))
			case int32:
				field.ReflectValueOf(ctx, value).SetInt(int64(data))
			case uint:
				field.ReflectValueOf(ctx, value).SetInt(int64(data))
			case uint8:
				field.ReflectValueOf(ctx, value).SetInt(int64(data))
			case uint16:
				field.ReflectValueOf(ctx, value).SetInt(int64(data))
			case uint32:
				field.ReflectValueOf(ctx, value).SetInt(int64(data))
			case uint64:
				field.ReflectValueOf(ctx, value).SetInt(int64(data))
			case float32:
				field.ReflectValueOf(ctx, value).SetInt(int64(data))
			case float64:
				field.ReflectValueOf(ctx, value).SetInt(int64(data))
			case []byte:
				return field.Set(ctx, value, string(data))
			case string:
				if i, err := strconv.ParseInt(data, 0, 64); err == nil {
					field.ReflectValueOf(ctx, value).SetInt(i)
				} else {
					return err
				}
			case time.Time:
				if field.AutoCreateTime == UnixNanosecond || field.AutoUpdateTime == UnixNanosecond {
					field.ReflectValueOf(ctx, value).SetInt(data.UnixNano())
				} else if field.AutoCreateTime == UnixMillisecond || field.AutoUpdateTime == UnixMillisecond {
					field.ReflectValueOf(ctx, value).SetInt(data.UnixMilli())
				} else {
					field.ReflectValueOf(ctx, value).SetInt(data.Unix())
				}
			case *time.Time:
				if data != nil {
					if field.AutoCreateTime == UnixNanosecond || field.AutoUpdateTime == UnixNanosecond {
						field.ReflectValueOf(ctx, value).SetInt(data.UnixNano())
					} else if field.AutoCreateTime == UnixMillisecond || field.AutoUpdateTime == UnixMillisecond {
						field.ReflectValueOf(ctx, value).SetInt(data.UnixMilli())
					} else {
						field.ReflectValueOf(ctx, value).SetInt(data.Unix())
					}
				} else {
					field.ReflectValueOf(ctx, value).SetInt(0)
				}
			default:
				return fallbackSetter(ctx, value, v, field.Set)
			}
			return err
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		field.Set = func(ctx context.Context, value reflect.Value, v interface{}) (err error) {
			switch data := v.(type) {
			case **uint64:
				if data != nil && *data != nil {
					field.ReflectValueOf(ctx, value).SetUint(**data)
				}
			case **uint:
				if data != nil && *data != nil {
					field.ReflectValueOf(ctx, value).SetUint(uint64(**data))
				}
			case **uint8:
				if data != nil && *data != nil {
					field.ReflectValueOf(ctx, value).SetUint(uint64(**data))
				}
			case **uint16:
				if data != nil && *data != nil {
					field.ReflectValueOf(ctx, value).SetUint(uint64(**data))
				}
			case **uint32:
				if data != nil && *data != nil {
					field.ReflectValueOf(ctx, value).SetUint(uint64(**data))
				}
			case uint64:
				field.ReflectValueOf(ctx, value).SetUint(data)
			case uint:
				field.ReflectValueOf(ctx, value).SetUint(uint64(data))
			case uint8:
				field.ReflectValueOf(ctx, value).SetUint(uint64(data))
			case uint16:
				field.ReflectValueOf(ctx, value).SetUint(uint64(data))
			case uint32:
				field.ReflectValueOf(ctx, value).SetUint(uint64(data))
			case int64:
				field.ReflectValueOf(ctx, value).SetUint(uint64(data))
			case int:
				field.ReflectValueOf(ctx, value).SetUint(uint64(data))
			case int8:
				field.ReflectValueOf(ctx, value).SetUint(uint64(data))
			case int16:
				field.ReflectValueOf(ctx, value).SetUint(uint64(data))
			case int32:
				field.ReflectValueOf(ctx, value).SetUint(uint64(data))
			case float32:
				field.ReflectValueOf(ctx, value).SetUint(uint64(data))
			case float64:
				field.ReflectValueOf(ctx, value).SetUint(uint64(data))
			case []byte:
				return field.Set(ctx, value, string(data))
			case time.Time:
				if field.AutoCreateTime == UnixNanosecond || field.AutoUpdateTime == UnixNanosecond {
					field.ReflectValueOf(ctx, value).SetUint(uint64(data.UnixNano()))
				} else if field.AutoCreateTime == UnixMillisecond || field.AutoUpdateTime == UnixMillisecond {
					field.ReflectValueOf(ctx, value).SetUint(uint64(data.UnixMilli()))
				} else {
					field.ReflectValueOf(ctx, value).SetUint(uint64(data.Unix()))
				}
			case string:
				if i, err := strconv.ParseUint(data, 0, 64); err == nil {
					field.ReflectValueOf(ctx, value).SetUint(i)
				} else {
					return err
				}
			default:
				return fallbackSetter(ctx, value, v, field.Set)
			}
			return err
		}
	case reflect.Float32, reflect.Float64:
		field.Set = func(ctx context.Context, value reflect.Value, v interface{}) (err error) {
			switch data := v.(type) {
			case **float64:
				if data != nil && *data != nil {
					field.ReflectValueOf(ctx, value).SetFloat(**data)
				}
			case **float32:
				if data != nil && *data != nil {
					field.ReflectValueOf(ctx, value).SetFloat(float64(**data))
				}
			case float64:
				field.ReflectValueOf(ctx, value).SetFloat(data)
			case float32:
				field.ReflectValueOf(ctx, value).SetFloat(float64(data))
			case int64:
				field.ReflectValueOf(ctx, value).SetFloat(float64(data))
			case int:
				field.ReflectValueOf(ctx, value).SetFloat(float64(data))
			case int8:
				field.ReflectValueOf(ctx, value).SetFloat(float64(data))
			case int16:
				field.ReflectValueOf(ctx, value).SetFloat(float64(data))
			case int32:
				field.ReflectValueOf(ctx, value).SetFloat(float64(data))
			case uint:
				field.ReflectValueOf(ctx, value).SetFloat(float64(data))
			case uint8:
				field.ReflectValueOf(ctx, value).SetFloat(float64(data))
			case uint16:
				field.ReflectValueOf(ctx, value).SetFloat(float64(data))
			case uint32:
				field.ReflectValueOf(ctx, value).SetFloat(float64(data))
			case uint64:
				field.ReflectValueOf(ctx, value).SetFloat(float64(data))
			case []byte:
				return field.Set(ctx, value, string(data))
			case string:
				if i, err := strconv.ParseFloat(data, 64); err == nil {
					field.ReflectValueOf(ctx, value).SetFloat(i)
				} else {
					return err
				}
			default:
				return fallbackSetter(ctx, value, v, field.Set)
			}
			return err
		}
	case reflect.String:
		field.Set = func(ctx context.Context, value reflect.Value, v interface{}) (err error) {
			switch data := v.(type) {
			case **string:
				if data != nil && *data != nil {
					field.ReflectValueOf(ctx, value).SetString(**data)
				}
			case string:
				field.ReflectValueOf(ctx, value).SetString(data)
			case []byte:
				field.ReflectValueOf(ctx, value).SetString(string(data))
			case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
				field.ReflectValueOf(ctx, value).SetString(utils.ToString(data))
			case float64, float32:
				field.ReflectValueOf(ctx, value).SetString(fmt.Sprintf("%."+strconv.Itoa(field.Precision)+"f", data))
			default:
				return fallbackSetter(ctx, value, v, field.Set)
			}
			return err
		}
	default:
		fieldValue := reflect.New(field.FieldType)
		switch fieldValue.Elem().Interface().(type) {
		case time.Time:
			field.Set = func(ctx context.Context, value reflect.Value, v interface{}) error {
				switch data := v.(type) {
				case **time.Time:
					if data != nil && *data != nil {
						field.Set(ctx, value, *data)
					}
				case time.Time:
					field.ReflectValueOf(ctx, value).Set(reflect.ValueOf(v))
				case *time.Time:
					if data != nil {
						field.ReflectValueOf(ctx, value).Set(reflect.ValueOf(data).Elem())
					} else {
						field.ReflectValueOf(ctx, value).Set(reflect.ValueOf(time.Time{}))
					}
				case string:
					if t, err := now.Parse(data); err == nil {
						field.ReflectValueOf(ctx, value).Set(reflect.ValueOf(t))
					} else {
						return fmt.Errorf("failed to set string %v to time.Time field %s, failed to parse it as time, got error %v", v, field.Name, err)
					}
				default:
					return fallbackSetter(ctx, value, v, field.Set)
				}
				return nil
			}
		case *time.Time:
			field.Set = func(ctx context.Context, value reflect.Value, v interface{}) error {
				switch data := v.(type) {
				case **time.Time:
					if data != nil && *data != nil {
						field.ReflectValueOf(ctx, value).Set(reflect.ValueOf(*data))
					}
				case time.Time:
					fieldValue := field.ReflectValueOf(ctx, value)
					if fieldValue.IsNil() {
						fieldValue.Set(reflect.New(field.FieldType.Elem()))
					}
					fieldValue.Elem().Set(reflect.ValueOf(v))
				case *time.Time:
					field.ReflectValueOf(ctx, value).Set(reflect.ValueOf(v))
				case string:
					if t, err := now.Parse(data); err == nil {
						fieldValue := field.ReflectValueOf(ctx, value)
						if fieldValue.IsNil() {
							if v == "" {
								return nil
							}
							fieldValue.Set(reflect.New(field.FieldType.Elem()))
						}
						fieldValue.Elem().Set(reflect.ValueOf(t))
					} else {
						return fmt.Errorf("failed to set string %v to time.Time field %s, failed to parse it as time, got error %v", v, field.Name, err)
					}
				default:
					return fallbackSetter(ctx, value, v, field.Set)
				}
				return nil
			}
		default:
			if _, ok := fieldValue.Elem().Interface().(sql.Scanner); ok {
				// pointer scanner
				field.Set = func(ctx context.Context, value reflect.Value, v interface{}) (err error) {
					reflectV := reflect.ValueOf(v)
					if !reflectV.IsValid() {
						field.ReflectValueOf(ctx, value).Set(reflect.New(field.FieldType).Elem())
					} else if reflectV.Kind() == reflect.Ptr && reflectV.IsNil() {
						return
					} else if reflectV.Type().AssignableTo(field.FieldType) {
						field.ReflectValueOf(ctx, value).Set(reflectV)
					} else if reflectV.Kind() == reflect.Ptr {
						return field.Set(ctx, value, reflectV.Elem().Interface())
					} else {
						fieldValue := field.ReflectValueOf(ctx, value)
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
				field.Set = func(ctx context.Context, value reflect.Value, v interface{}) (err error) {
					reflectV := reflect.ValueOf(v)
					if !reflectV.IsValid() {
						field.ReflectValueOf(ctx, value).Set(reflect.New(field.FieldType).Elem())
					} else if reflectV.Kind() == reflect.Ptr && reflectV.IsNil() {
						return
					} else if reflectV.Type().AssignableTo(field.FieldType) {
						field.ReflectValueOf(ctx, value).Set(reflectV)
					} else if reflectV.Kind() == reflect.Ptr {
						return field.Set(ctx, value, reflectV.Elem().Interface())
					} else {
						if valuer, ok := v.(driver.Valuer); ok {
							v, _ = valuer.Value()
						}

						err = field.ReflectValueOf(ctx, value).Addr().Interface().(sql.Scanner).Scan(v)
					}
					return
				}
			} else {
				field.Set = func(ctx context.Context, value reflect.Value, v interface{}) (err error) {
					return fallbackSetter(ctx, value, v, field.Set)
				}
			}
		}
	}

	if field.Serializer != nil {
		var (
			oldFieldSetter = field.Set
			sameElemType   bool
			sameType       = field.FieldType == reflect.ValueOf(field.Serializer).Type()
		)

		if reflect.ValueOf(field.Serializer).Kind() == reflect.Ptr {
			sameElemType = field.FieldType == reflect.ValueOf(field.Serializer).Type().Elem()
		}

		serializerValue := reflect.Indirect(reflect.ValueOf(field.Serializer))
		serializerType := serializerValue.Type()
		field.Set = func(ctx context.Context, value reflect.Value, v interface{}) (err error) {
			if s, ok := v.(*serializer); ok {
				if s.fieldValue == nil && s.Serializer == nil {
					rv := field.ReflectValueOf(ctx, value)
					if rv.IsValid() && rv.CanSet() {
						rv.Set(reflect.Zero(field.FieldType))
					}
					return nil
				}
				if s.fieldValue != nil {
					err = oldFieldSetter(ctx, value, s.fieldValue)
				} else if err = s.Serializer.Scan(ctx, field, value, s.value); err == nil {
					if sameElemType {
						field.ReflectValueOf(ctx, value).Set(reflect.ValueOf(s.Serializer).Elem())
					} else if sameType {
						field.ReflectValueOf(ctx, value).Set(reflect.ValueOf(s.Serializer))
					}
					si := reflect.New(serializerType)
					si.Elem().Set(serializerValue)
					s.Serializer = si.Interface().(SerializerInterface)
				}
			} else {
				err = oldFieldSetter(ctx, value, v)
			}
			return
		}
	}
}

// setupNewValuePool installs a field-specific pool of serializer wrappers when a codec
// is present; otherwise it reuses a type-keyed pool of pointers to the indirect field
// type. Serializer instances are shallow-copied from the configured prototype, so
// custom codecs must manage any shared referenced state safely.
func (field *Field) setupNewValuePool() {
	if field.Serializer != nil {
		serializerValue := reflect.Indirect(reflect.ValueOf(field.Serializer))
		serializerType := serializerValue.Type()
		field.NewValuePool = &sync.Pool{
			New: func() interface{} {
				si := reflect.New(serializerType)
				si.Elem().Set(serializerValue)
				return &serializer{
					Field:      field,
					Serializer: si.Interface().(SerializerInterface),
				}
			},
		}
	}

	if field.NewValuePool == nil {
		field.NewValuePool = poolInitializer(reflect.PointerTo(field.IndirectFieldType))
	}
}
