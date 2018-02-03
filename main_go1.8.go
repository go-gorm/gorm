// +build go1.8

package gorm

import (
	"context"
	"database/sql"
)

// WithContext specify context to be passed to the underlying `*sql.DB` or
// `*sql.Tx` query methods
func (s *DB) WithContext(ctx context.Context) *DB {
	db := s.clone()
	db.context = ctx
	return db
}

// Context returns the specified context for this instance, or nil if not set
func (s *DB) Context() context.Context {
	return s.context
}

func (s *DB) contextOrBackground() context.Context {
	if s.context != nil {
		return s.context
	}
	return context.Background()
}

// BeginTx starts a transaction with the given options
func (s *DB) BeginTx(opts *sql.TxOptions) *DB {
	c := s.clone()
	if db, ok := c.db.(sqlDb); ok && db != nil {
		tx, err := db.BeginTx(s.contextOrBackground(), opts)
		c.db = interface{}(tx).(SQLCommon)
		c.AddError(err)
	} else {
		c.AddError(ErrCantStartTransaction)
	}
	return c
}

// Begin starts a transaction
func (s *DB) Begin() *DB {
	return s.BeginTx(nil)
}
