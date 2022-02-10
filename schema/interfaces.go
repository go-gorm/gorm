package schema

import (
	"context"
	"reflect"

	"gorm.io/gorm/clause"
)

// GormDataTypeInterface gorm data type interface
type GormDataTypeInterface interface {
	GormDataType() string
}

// Serializer serializer interface
type Serializer interface {
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
