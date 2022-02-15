package schema

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"gorm.io/gorm/clause"
)

// GormDataTypeInterface gorm data type interface
type GormDataTypeInterface interface {
	GormDataType() string
}

// FieldNewValuePool field new scan value pool
type FieldNewValuePool interface {
	Get() interface{}
	Put(interface{})
}

// Serializer field value serializer
type Serializer struct {
	Field       *Field
	Interface   SerializerInterface
	Destination reflect.Value
	Context     context.Context
	value       interface{}
}

// Scan implements sql.Scanner interface
func (s *Serializer) Scan(value interface{}) error {
	s.value = value
	return nil
}

// Value implements driver.Valuer interface
func (s Serializer) Value() (driver.Value, error) {
	return s.Interface.Value(s.Context, s.Field, s.Destination)
}

// SerializerInterface serializer interface
type SerializerInterface interface {
	Scan(ctx context.Context, field *Field, dst reflect.Value, dbValue interface{}) error
	Value(ctx context.Context, field *Field, dst reflect.Value) (interface{}, error)
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
func (JSONSerializer) Value(ctx context.Context, field *Field, dst reflect.Value) (interface{}, error) {
	fv, _ := field.ValueOf(ctx, dst)
	return fv, nil
}

// CreateClausesInterface create clauses interface
type CreateClausesInterface interface {
	CreateClauses(*Field) []clause.Interface
}

// QueryClausesInterface query clauses interface
type QueryClausesInterface interface {
	QueryClauses(*Field) []clause.Interface
}

// UpdateClausesInterface update clauses interface
type UpdateClausesInterface interface {
	UpdateClauses(*Field) []clause.Interface
}

// DeleteClausesInterface delete clauses interface
type DeleteClausesInterface interface {
	DeleteClauses(*Field) []clause.Interface
}
