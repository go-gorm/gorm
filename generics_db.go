//go:build go1.27

package gorm

import "gorm.io/gorm/clause"

// G returns a type-safe generic query interface backed by db.
//
// This method is available when building with Go 1.27 or newer. For older Go
// versions, use the package-level G function instead.
func (db *DB) G[T any](opts ...clause.Expression) Interface[T] {
	return G[T](db, opts...)
}
