package model

import "errors"

var (
	// ErrInvalidTable invalid table name
	ErrInvalidTable = errors.New("invalid table name")
	// ErrUnaddressable unaddressable value
	ErrUnaddressable = errors.New("using unaddressable value")
)
