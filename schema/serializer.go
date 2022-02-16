package schema

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"
)

var serializerMap = sync.Map{}

// RegisterSerializer register serializer
func RegisterSerializer(name string, serializer SerializerInterface) {
	serializerMap.Store(strings.ToLower(name), serializer)
}

// GetSerializer get serializer
func GetSerializer(name string) (serializer SerializerInterface, ok bool) {
	v, ok := serializerMap.Load(strings.ToLower(name))
	if ok {
		serializer, ok = v.(SerializerInterface)
	}
	return serializer, ok
}

func init() {
	RegisterSerializer("json", JSONSerializer{})
	RegisterSerializer("unixtime", UnixSecondSerializer{})
}

// Serializer field value serializer
type serializer struct {
	Field           *Field
	Serializer      SerializerInterface
	SerializeValuer SerializerValuerInterface
	Destination     reflect.Value
	Context         context.Context
	value           interface{}
	fieldValue      interface{}
}

// Scan implements sql.Scanner interface
func (s *serializer) Scan(value interface{}) error {
	s.value = value
	return nil
}

// Value implements driver.Valuer interface
func (s serializer) Value() (driver.Value, error) {
	return s.SerializeValuer.Value(s.Context, s.Field, s.Destination, s.fieldValue)
}

// SerializerInterface serializer interface
type SerializerInterface interface {
	Scan(ctx context.Context, field *Field, dst reflect.Value, dbValue interface{}) error
	SerializerValuerInterface
}

// SerializerValuerInterface serializer valuer interface
type SerializerValuerInterface interface {
	Value(ctx context.Context, field *Field, dst reflect.Value, fieldValue interface{}) (interface{}, error)
}

// JSONSerializer json serializer
type JSONSerializer struct {
}

// Scan implements serializer interface
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
			return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", dbValue))
		}

		err = json.Unmarshal(bytes, fieldValue.Interface())
	}

	field.ReflectValueOf(ctx, dst).Set(fieldValue.Elem())
	return
}

// Value implements serializer interface
func (JSONSerializer) Value(ctx context.Context, field *Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	result, err := json.Marshal(fieldValue)
	return string(result), err
}

// UnixSecondSerializer json serializer
type UnixSecondSerializer struct {
}

// Scan implements serializer interface
func (UnixSecondSerializer) Scan(ctx context.Context, field *Field, dst reflect.Value, dbValue interface{}) (err error) {
	t := sql.NullTime{}
	if err = t.Scan(dbValue); err == nil {
		err = field.Set(ctx, dst, t.Time)
	}

	return
}

// Value implements serializer interface
func (UnixSecondSerializer) Value(ctx context.Context, field *Field, dst reflect.Value, fieldValue interface{}) (result interface{}, err error) {
	switch v := fieldValue.(type) {
	case int64, int, uint, uint64, int32, uint32, int16, uint16:
		result = time.Unix(reflect.ValueOf(v).Int(), 0)
	default:
		err = fmt.Errorf("invalid field type %#v for UnixSecondSerializer, only int, uint supported", v)
	}
	return
}
