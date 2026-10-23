package gorm

import (
	"errors"

	"gorm.io/gorm/logger"
)

var (
	// ErrRecordNotFound indicates that First, Take, or Last found no matching row.
	// Use errors.Is to test it; Find normally reports an empty result without this error.
	ErrRecordNotFound = logger.ErrRecordNotFound
	// ErrInvalidTransaction indicates an operation requiring a transaction-capable
	// connection was attempted on an unsupported or invalid connection. Check Begin
	// before issuing work, and call Commit or Rollback on the returned transaction.
	ErrInvalidTransaction = errors.New("invalid transaction")
	// ErrNotImplemented is the sentinel for an operation with no implementation.
	// Support depends on the component or dialector reporting it.
	ErrNotImplemented = errors.New("not implemented")
	// ErrMissingWhereClause indicates an UPDATE or DELETE was blocked because
	// no user condition restricted it. Add a deliberate predicate; only enable
	// AllowGlobalUpdate when affecting all rows is intended. Raw Exec bypasses this guard.
	ErrMissingWhereClause = errors.New("WHERE conditions required")
	// ErrUnsupportedRelation indicates that an association name is not present
	// in the parsed model relationships. Use the Go association field name or path,
	// not a database table name; callers may wrap this error with the offending name.
	ErrUnsupportedRelation = errors.New("unsupported relations")
	// ErrPrimaryKeyRequired indicates an operation requires primary-key values
	// that are absent, such as continuing FindInBatches or changing associations.
	// Include primary keys in projections and supply persisted owner records.
	ErrPrimaryKeyRequired = errors.New("primary key required")
	// ErrModelValueRequired indicates SQL construction needs a model schema.
	// Supply Model or a model destination rather than only an untyped table expression.
	ErrModelValueRequired = errors.New("model value required")
	// ErrModelAccessibleFieldsRequired indicates that a model has no accessible
	// fields for a schema-dependent expression, such as the primary-key fallback.
	ErrModelAccessibleFieldsRequired = errors.New("model accessible fields required")
	// ErrSubQueryRequired indicates that creating a view requires a non-nil
	// ViewOption.Query. Supply a query builder describing the view selection.
	ErrSubQueryRequired = errors.New("sub query required")
	// ErrInvalidData indicates that an operation could not interpret or assign
	// the supplied data. Check the destination shape and field types for that operation.
	ErrInvalidData = errors.New("unsupported data")
	// ErrUnsupportedDriver indicates the active dialector lacks an optional
	// capability required by the operation, such as savepoints.
	ErrUnsupportedDriver = errors.New("unsupported driver")
	// ErrRegistered indicates Use was called with a plugin name already present
	// in the shared registry. A plugin name must be unique within the DB configuration.
	ErrRegistered = errors.New("registered")
	// ErrInvalidField indicates a requested model field cannot be found or
	// assigned, for example by Statement.SetColumn. Check schema mapping and field names.
	ErrInvalidField = errors.New("invalid field")
	// ErrEmptySlice indicates a create operation received an empty input slice
	// where records were required. Handle empty application batches before insertion.
	ErrEmptySlice = errors.New("empty slice found")
	// ErrDryRunModeUnsupported indicates an operation needs actual result rows
	// that DryRun cannot produce. Inspect Statement.SQL and Vars for SQL generation.
	ErrDryRunModeUnsupported = errors.New("dry run mode unsupported")
	// ErrInvalidDB indicates that the current connection cannot provide a valid
	// *sql.DB, or a prepared-statement wrapper can no longer use its store.
	ErrInvalidDB = errors.New("invalid db")
	// ErrInvalidValue indicates the supplied model or destination has an invalid
	// shape or cannot be assigned. Query destinations generally need writable pointers
	// to supported structs, slices, maps, or scalar values.
	ErrInvalidValue = errors.New("invalid value, should be pointer to struct or slice")
	// ErrInvalidValueOfLength indicates batch association values do not match
	// the number of owner records. Provide one appropriately shaped argument per owner.
	ErrInvalidValueOfLength = errors.New("invalid association values, length doesn't match")
	// ErrPreloadNotAllowed is the sentinel for a preload request rejected in a
	// count context. Count returns an aggregate rather than model rows to preload.
	ErrPreloadNotAllowed = errors.New("preload is not allowed when count is used")
	// ErrDuplicatedKey represents a unique-constraint violation translated by a
	// supporting dialector when TranslateError is enabled. Otherwise inspect the native
	// driver error. Use errors.Is rather than matching the error text.
	ErrDuplicatedKey = errors.New("duplicated key not allowed")
	// ErrForeignKeyViolated represents a foreign-key violation translated by a
	// supporting dialector when TranslateError is enabled. Native errors remain possible
	// when translation is disabled or the driver does not recognize the error.
	ErrForeignKeyViolated = errors.New("violates foreign key constraint")
	// ErrCheckConstraintViolated represents a check-constraint violation translated
	// by a supporting dialector when TranslateError is enabled. Translation coverage
	// depends on the driver; use errors.Is for this sentinel.
	ErrCheckConstraintViolated = errors.New("violates check constraint")
)
