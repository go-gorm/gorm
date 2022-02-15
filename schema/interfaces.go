package schema

import (
	"context"
	"database/sql/driver"
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

type fieldNewValuePool struct {
	getter func() interface{}
	putter func(interface{})
}

func (fp fieldNewValuePool) Get() interface{} {
	return fp.getter()
}

func (fp fieldNewValuePool) Put(v interface{}) {
	fp.putter(v)
}

// Serializer field value serializer
type Serializer struct {
	Field       *Field
	Interface   SerializerInterface
	Destination reflect.Value
	Context     context.Context
}

// Scan implements sql.Scanner interface
func (s *Serializer) Scan(value interface{}) error {
	return s.Interface.Scan(s.Context, s.Field, s.Destination, value)
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
