package schema

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strings"
	"sync"
	"time"
)

// serializerMap stores serializer prototypes under lowercase names for concurrent
// registration and lookup.
var serializerMap = sync.Map{}

// RegisterSerializer installs or replaces a serializer under a case-insensitive name,
// used by gorm:"serializer:name" tags. Register before parsing models: cached Field
// metadata keeps its selected serializer and does not refresh after a replacement. A
// codec must support the field types that select it and safely handle any shared state.
func RegisterSerializer(name string, serializer SerializerInterface) {
	serializerMap.Store(strings.ToLower(name), serializer)
}

// GetSerializer looks up a case-insensitive registered name and reports whether a
// SerializerInterface was found. Built-ins are json, gob, and unixtime; the returned
// codec is the registered prototype, not a per-call deep copy.
func GetSerializer(name string) (serializer SerializerInterface, ok bool) {
	v, ok := serializerMap.Load(strings.ToLower(name))
	if ok {
		serializer, ok = v.(SerializerInterface)
	}
	return serializer, ok
}

// init registers the built-in json, unixtime, and gob serializers before model parsing.
func init() {
	RegisterSerializer("json", JSONSerializer{})
	RegisterSerializer("unixtime", UnixSecondSerializer{})
	RegisterSerializer("gob", GobSerializer{})
}

// serializer bridges database/sql scanning and valuation to a field codec. Scan
// captures the raw database value; Field.Set later invokes the codec with the model
// destination. Instances used for scanning are pooled and must not escape their borrow
// lifetime.
type serializer struct {
	// Field identifies the model field and its metadata for codec calls.
	Field *Field
	// Serializer is the scan codec instance copied from the field prototype.
	Serializer SerializerInterface
	// SerializeValuer is the write codec, selected from the field value when it implements
	// SerializerValuerInterface or from the field's configured serializer.
	SerializeValuer SerializerValuerInterface
	// Destination is the enclosing model value supplied to serialization.
	Destination reflect.Value
	// Context is the operation context passed to the write codec.
	Context context.Context
	// value holds the raw database value captured by Scan until Field.Set performs
	// decoding.
	value interface{}
	// fieldValue holds a Go field value for writes or direct assignment, avoiding a decode
	// when already provided.
	fieldValue interface{}
}

// Scan implements sql.Scanner by retaining value for deferred decoding in Field.Set. It
// returns nil without decoding or assigning the model.
func (s *serializer) Scan(value interface{}) error {
	s.value = value
	return nil
}

// Value implements driver.Valuer by delegating to SerializeValuer with the captured
// context, field metadata, model, and field value. The codec must return a
// database-driver-compatible value.
func (s serializer) Value() (driver.Value, error) {
	return s.SerializeValuer.Value(s.Context, s.Field, s.Destination, s.fieldValue)
}

// SerializerInterface converts a model field between Go values and database values.
// Implement it on a field type or register a prototype and select it with
// gorm:"serializer:name".
//
// Scan receives the enclosing model, not the field alone, and should assign through
// field.ReflectValueOf or field.Set as appropriate. Custom stateful codecs should
// account for shallow prototype copies and concurrent writes; mutable maps or pointers
// in a prototype are not deep-copied.
type SerializerInterface interface {
	// Scan decodes dbValue into the field of addressable model dst using ctx. It must
	// handle the database's value types and NULL policy, and return decoding or assignment
	// errors.
	Scan(ctx context.Context, field *Field, dst reflect.Value, dbValue interface{}) error
	// SerializerValuerInterface supplies the corresponding write-side conversion.
	SerializerValuerInterface
}

// SerializerValuerInterface converts a Go field value to a database parameter. Field
// values can implement it to override the configured serializer's write behavior while
// retaining that serializer for reads.
type SerializerValuerInterface interface {
	// Value serializes fieldValue from model dst using ctx and field metadata. Return a
	// driver-compatible value, nil for SQL NULL, or an error; the returned value is bound
	// as data rather than SQL text.
	Value(ctx context.Context, field *Field, dst reflect.Value, fieldValue interface{}) (interface{}, error)
}

// JSONSerializer is the built-in codec selected by gorm:"serializer:json". It uses
// encoding/json for field values. Serializer setup initially supplies a string data
// type, but subsequent native field-type inference, GormDataType, or a type tag can
// override it. Specify a compatible type tag when a particular JSON/text database
// column is required. A []byte field follows encoding/json rules, including base64
// representation.
type JSONSerializer struct{}

// Scan decodes JSON from string or []byte; other non-nil database values are
// JSON-marshaled first. SQL NULL and empty input assign the field's zero value. It
// writes the decoded value to dst even when Unmarshal returns an error, so callers must
// check the error.
func (JSONSerializer) Scan(ctx context.Context, field *Field, dst reflect.Value, dbValue interface{}) (err error) {
	fieldValue := reflect.New(field.FieldType)

	if dbValue != nil {
		var bytes []byte
		switch v := dbValue.(type) {
		case []byte:
			bytes = v
		case string:
			bytes = []byte(v)
		default:
			bytes, err = json.Marshal(v)
			if err != nil {
				return err
			}
		}

		if len(bytes) > 0 {
			err = json.Unmarshal(bytes, fieldValue.Interface())
		}
	}

	field.ReflectValueOf(ctx, dst).Set(fieldValue.Elem())
	return
}

// Value JSON-marshals fieldValue and returns JSON text. A JSON null becomes SQL NULL,
// except a NOT NULL tag produces an empty string; marshaling errors are returned. A
// notNull tag is schema metadata and should not be confused with that exact NOT NULL
// serializer setting.
func (JSONSerializer) Value(ctx context.Context, field *Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	result, err := json.Marshal(fieldValue)
	if string(result) == "null" {
		if field.TagSettings["NOT NULL"] != "" {
			return "", nil
		}
		return nil, err
	}
	return string(result), err
}

// UnixSecondSerializer is the built-in unixtime codec for integer Go fields backed by
// database time values. For example, tag an int64 with
// gorm:"serializer:unixtime;type:time". It stores epoch seconds as UTC time.Time,
// rather than storing the integer directly.
type UnixSecondSerializer struct{}

// Scan converts a database value through sql.NullTime and assigns Unix seconds when
// valid. SQL NULL leaves the destination unchanged; incompatible database values return
// the sql.NullTime scan error.
func (UnixSecondSerializer) Scan(ctx context.Context, field *Field, dst reflect.Value, dbValue interface{}) (err error) {
	t := sql.NullTime{}
	if err = t.Scan(dbValue); err == nil && t.Valid {
		err = field.Set(ctx, dst, t.Time.Unix())
	}

	return
}

// Value converts built-in signed or unsigned integer values, or pointers to them, to
// UTC time.Time at Unix-second precision. Nil integer pointers become SQL NULL; uint
// values above MaxInt64 and unsupported types return errors.
func (UnixSecondSerializer) Value(ctx context.Context, field *Field, dst reflect.Value, fieldValue interface{}) (result interface{}, err error) {
	rv := reflect.ValueOf(fieldValue)
	switch fieldValue.(type) {
	case int, int8, int16, int32, int64:
		result = time.Unix(rv.Int(), 0).UTC()
	case uint, uint8, uint16, uint32, uint64:
		if uv := rv.Uint(); uv > math.MaxInt64 {
			err = fmt.Errorf("integer overflow conversion uint64(%d) -> int64", uv)
		} else {
			result = time.Unix(int64(uv), 0).UTC() //nolint:gosec
		}
	case *int, *int8, *int16, *int32, *int64:
		if rv.IsZero() {
			return nil, nil
		}
		result = time.Unix(rv.Elem().Int(), 0).UTC()
	case *uint, *uint8, *uint16, *uint32, *uint64:
		if rv.IsZero() {
			return nil, nil
		}
		if uv := rv.Elem().Uint(); uv > math.MaxInt64 {
			err = fmt.Errorf("integer overflow conversion uint64(%d) -> int64", uv)
		} else {
			result = time.Unix(int64(uv), 0).UTC() //nolint:gosec
		}
	default:
		err = fmt.Errorf("invalid field type %#v for UnixSecondSerializer, only int, uint supported", fieldValue)
	}
	return
}

// GobSerializer is the built-in gob codec. Use a compatible binary column, for example
// gorm:"serializer:gob;type:bytes", and follow encoding/gob type and registration
// requirements.
type GobSerializer struct{}

// Scan decodes a []byte database value using encoding/gob. SQL NULL or empty bytes
// assign the field's zero value; other database value types return an error. A decode
// error can still leave a partially decoded destination, so callers must check it.
func (GobSerializer) Scan(ctx context.Context, field *Field, dst reflect.Value, dbValue interface{}) (err error) {
	fieldValue := reflect.New(field.FieldType)

	if dbValue != nil {
		var bytesValue []byte
		switch v := dbValue.(type) {
		case []byte:
			bytesValue = v
		default:
			return fmt.Errorf("failed to unmarshal gob value: %#v", dbValue)
		}
		if len(bytesValue) > 0 {
			decoder := gob.NewDecoder(bytes.NewBuffer(bytesValue))
			err = decoder.Decode(fieldValue.Interface())
		}
	}
	field.ReflectValueOf(ctx, dst).Set(fieldValue.Elem())
	return
}

// Value gob-encodes fieldValue and returns the resulting bytes and any encoding error.
func (GobSerializer) Value(ctx context.Context, field *Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	buf := new(bytes.Buffer)
	err := gob.NewEncoder(buf).Encode(fieldValue)
	return buf.Bytes(), err
}
