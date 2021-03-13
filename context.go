package gorm

import (
	"context"
)

// contextKeyType is an unexported type so that the context key never
// collides with any other context keys.
type contextKeyType struct{}

// contextKey is the key used for the context to store the DB object.
var contextKey = contextKeyType{}

// WithContext inserts a DB into the context and is retrievable using FromContext().
func WithContext(ctx context.Context, db *DB) context.Context {
	return context.WithValue(ctx, contextKey, db)
}

// FromContext extracts a DB from the context. An error is returned if
// the context does not contain a DB object.
func FromContext(ctx context.Context) (*DB, error) {
	db, _ := ctx.Value(contextKey).(*DB)
	if db == nil {
		return nil, ErrDBNotFoundInContext
	}
	return db, nil
}
