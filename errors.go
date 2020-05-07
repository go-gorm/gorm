package gorm

import (
	"errors"
)

var (
	// ErrRecordNotFound record not found error
	ErrRecordNotFound = errors.New("record not found")
	// ErrInvalidSQL invalid SQL error, happens when you passed invalid SQL
	ErrInvalidSQL = errors.New("invalid SQL")
	// ErrInvalidTransaction invalid transaction when you are trying to `Commit` or `Rollback`
	ErrInvalidTransaction = errors.New("no valid transaction")
	// ErrUnaddressable unaddressable value
	ErrUnaddressable = errors.New("using unaddressable value")
	// ErrNotImplemented not implemented
	ErrNotImplemented = errors.New("not implemented")
	// ErrMissingWhereClause missing where clause
	ErrMissingWhereClause = errors.New("missing WHERE clause while deleting")
	// ErrUnsupportedRelation unsupported relations
	ErrUnsupportedRelation = errors.New("unsupported relations")
)
