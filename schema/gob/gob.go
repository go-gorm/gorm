// Package gob provides the Gob-based serializer for GORM.
// Import it with a blank identifier to auto-register the serializer:
//
//	import _ "gorm.io/gorm/schema/gob"
//
// Or register it manually:
//
//	schema.RegisterSerializer("gob", gob.GobSerializer{})
package gob

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"reflect"

	"gorm.io/gorm/schema"
)

func init() {
	schema.RegisterSerializer("gob", GobSerializer{})
}

// GobSerializer gob serializer
type GobSerializer struct{}

// Scan implements serializer interface
func (GobSerializer) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) (err error) {
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
func (GobSerializer) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	buf := new(bytes.Buffer)
	err := gob.NewEncoder(buf).Encode(fieldValue)
	return buf.Bytes(), err
}
