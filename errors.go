package gorm

import (
	"errors"
)

var (
	// ErrRecordNotFound record not found error
	ErrRecordNotFound = errors.New("record not found")
	// ErrInvalidTransaction invalid transaction when you are trying to `Commit` or `Rollback`
	ErrInvalidTransaction = errors.New("no valid transaction")
	// ErrNotImplemented not implemented
	ErrNotImplemented = errors.New("not implemented")
	// ErrMissingWhereClause missing where clause
	ErrMissingWhereClause = errors.New("WHERE conditions required")
	// ErrUnsupportedRelation unsupported relations
	ErrUnsupportedRelation = errors.New("unsupported relations")
	// ErrorPrimaryKeyRequired primary keys required
	ErrorPrimaryKeyRequired = errors.New("primary key required")
	// ErrorModelValueRequired model value required
	ErrorModelValueRequired = errors.New("model value required")
	// ErrUnsupportedDriver unsupported driver
	ErrUnsupportedDriver = errors.New("unsupported driver")
	// ErrRegistered registered
	ErrRegistered = errors.New("registered")
	// ErrInvalidField invalid field
	ErrInvalidField = errors.New("invalid field")
)
