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
	RegisterSerializer("gob", GobSerializer{})
	RegisterSerializer("pgarray", PgArraySerializer{})
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
type JSONSerializer struct{}

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

// Value implements serializer interface
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

// UnixSecondSerializer json serializer
type UnixSecondSerializer struct{}

// Scan implements serializer interface
func (UnixSecondSerializer) Scan(ctx context.Context, field *Field, dst reflect.Value, dbValue interface{}) (err error) {
	t := sql.NullTime{}
	if err = t.Scan(dbValue); err == nil && t.Valid {
		err = field.Set(ctx, dst, t.Time.Unix())
	}

	return
}

// Value implements serializer interface
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

// GobSerializer gob serializer
type GobSerializer struct{}

// Scan implements serializer interface
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

// Value implements serializer interface
func (GobSerializer) Value(ctx context.Context, field *Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	buf := new(bytes.Buffer)
	err := gob.NewEncoder(buf).Encode(fieldValue)
	return buf.Bytes(), err
}

// PgArraySerializer PostgreSQL text[] array serializer
type PgArraySerializer struct{}

// Scan implements serializer interface
func (PgArraySerializer) Scan(ctx context.Context, field *Field, dst reflect.Value, dbValue interface{}) (err error) {
	fieldValue := reflect.New(field.FieldType)

	if dbValue != nil {
		var str string
		switch v := dbValue.(type) {
		case []byte:
			str = string(v)
		case string:
			str = v
		default:
			return fmt.Errorf("failed to parse pg array value: %#v", dbValue)
		}

		if len(str) > 0 {
			parsed, err := parsePgArray(str)
			if err != nil {
				return err
			}
			fieldValue.Elem().Set(reflect.ValueOf(parsed))
		}
	}

	field.ReflectValueOf(ctx, dst).Set(fieldValue.Elem())
	return
}

// Value implements serializer interface
func (PgArraySerializer) Value(ctx context.Context, field *Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	if fieldValue == nil {
		return nil, nil
	}

	rv := reflect.ValueOf(fieldValue)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, nil
		}
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Slice {
		return nil, fmt.Errorf("PgArraySerializer expects a slice, got %T", fieldValue)
	}

	if rv.Len() == 0 {
		return nil, nil
	}

	var buf strings.Builder
	buf.WriteString("{")

	for i := 0; i < rv.Len(); i++ {
		if i > 0 {
			buf.WriteString(",")
		}
		elem := rv.Index(i).Interface()
		str, ok := elem.(string)
		if !ok {
			return nil, fmt.Errorf("PgArraySerializer expects []string, got element of type %T", elem)
		}
		buf.WriteString(escapePgArrayElement(str))
	}

	buf.WriteString("}")
	return buf.String(), nil
}

// parsePgArray parses PostgreSQL array format: {elem1,elem2,"quoted,elem"}
func parsePgArray(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "{}" {
		return []string{}, nil
	}

	if len(s) < 2 || s[0] != '{' || s[len(s)-1] != '}' {
		return nil, fmt.Errorf("invalid pg array format: %s", s)
	}

	// Remove surrounding braces
	s = s[1 : len(s)-1]
	if s == "" {
		return []string{}, nil
	}

	var result []string
	var current strings.Builder
	inQuotes := false
	escaped := false

	for i := 0; i < len(s); i++ {
		c := s[i]

		if escaped {
			current.WriteByte(c)
			escaped = false
			continue
		}

		if c == '\\' {
			escaped = true
			continue
		}

		if c == '"' {
			inQuotes = !inQuotes
			continue
		}

		if c == ',' && !inQuotes {
			result = append(result, current.String())
			current.Reset()
			continue
		}

		current.WriteByte(c)
	}

	result = append(result, current.String())
	return result, nil
}

// escapePgArrayElement escapes a string for PostgreSQL array format
func escapePgArrayElement(s string) string {
	needsQuotes := false
	for _, c := range s {
		if c == ',' || c == '"' || c == '\\' || c == '{' || c == '}' || c == ' ' {
			needsQuotes = true
			break
		}
	}

	if !needsQuotes {
		return s
	}

	var buf strings.Builder
	buf.WriteString("\"")
	for _, c := range s {
		if c == '"' || c == '\\' {
			buf.WriteRune('\\')
		}
		buf.WriteRune(c)
	}
	buf.WriteString("\"")
	return buf.String()
}
