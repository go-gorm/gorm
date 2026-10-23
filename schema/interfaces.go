package schema

import (
	"gorm.io/gorm/clause"
)

// ConstraintInterface supplies a named SQL constraint definition to migrators.
// Implementations return a template plus clause arguments; the caller executes the
// resulting DDL.
type ConstraintInterface interface {
	// GetName returns the constraint identifier used for migration lookup and creation.
	GetName() string
	// Build returns the definition SQL template and its arguments. Represent identifiers
	// with clause.Column or clause.Table and reserve raw SQL for trusted schema
	// expressions.
	Build() (sql string, vars []interface{})
}

// GormDataTypeInterface lets a custom field type report its logical GORM type during
// parsing. A type tag may still override Field.DataType; dialect-specific migration
// types are a separate extension.
type GormDataTypeInterface interface {
	// GormDataType returns a stable logical type name such as json or string, without
	// inspecting a particular row value.
	GormDataType() string
}

// FieldNewValuePool supplies reusable scan destinations for a field. Implementations
// must support concurrent borrowing; a borrowed value belongs to the caller until Put
// and must not be retained afterward.
type FieldNewValuePool interface {
	// Get borrows a destination compatible with the field's scan and Set paths. The caller
	// must later return it with Put.
	Get() interface{}
	// Put returns a previously borrowed destination after its result has been assigned to
	// the model.
	Put(interface{})
}

// CreateClausesInterface lets a field type contribute clauses to the schema's insert
// callbacks. Parse calls it on a new field value and caches the clauses;
// implementations should derive behavior from Field metadata, not an individual model
// instance.
type CreateClausesInterface interface {
	// CreateClauses returns field-specific clauses used by insert callbacks. Return nil
	// when no customization is needed; clause implementations can modify a statement when
	// added.
	CreateClauses(*Field) []clause.Interface
}

// QueryClausesInterface lets a field type contribute clauses to the schema's query
// callbacks. Parse calls it on a new field value and caches the clauses;
// implementations should derive behavior from Field metadata, not an individual model
// instance.
type QueryClausesInterface interface {
	// QueryClauses returns field-specific clauses used by query callbacks. Return nil when
	// no customization is needed; clause implementations can modify a statement when added.
	QueryClauses(*Field) []clause.Interface
}

// UpdateClausesInterface lets a field type contribute clauses to the schema's update
// callbacks. Parse calls it on a new field value and caches the clauses;
// implementations should derive behavior from Field metadata, not an individual model
// instance.
type UpdateClausesInterface interface {
	// UpdateClauses returns field-specific clauses used by update callbacks. Return nil
	// when no customization is needed; clause implementations can modify a statement when
	// added.
	UpdateClauses(*Field) []clause.Interface
}

// DeleteClausesInterface lets a field type contribute clauses to the schema's delete
// callbacks. Parse calls it on a new field value and caches the clauses;
// implementations should derive behavior from Field metadata, not an individual model
// instance.
type DeleteClausesInterface interface {
	// DeleteClauses returns field-specific clauses used by delete callbacks. Return nil
	// when no customization is needed; clause implementations can modify a statement when
	// added.
	DeleteClauses(*Field) []clause.Interface
}
