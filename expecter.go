package gorm

import (
	"fmt"
)

// Expecter is the exported struct used for setting expectations
type Expecter struct {
	adapter Adapter
}

// AdapterFactory is a generic interface for arbitrary adapters that satisfy
// the interface. variadic args are passed to gorm.Open.
type AdapterFactory func(dialect string, args ...interface{}) (error, *DB, Adapter)

func (h *Expecter) ExpectFirst(model interface{}) ExpectedQuery {
	fmt.Printf("Expecting query: %s", "some query\n")
	return h.adapter.ExpectQuery("some sql")
}

func (h *Expecter) ExpectFind(model interface{}) ExpectedQuery {
	fmt.Println("Expecting query: %s", "some query involving Find")
	return h.adapter.ExpectQuery("some find condition")
}

// NewDefaultExpecter returns a Expecter powered by go-sqlmock
func NewDefaultExpecter() (error, *DB, *Expecter) {
	err, gormDb, adapter := NewSqlmockAdapter("sqlmock", "mock_gorm_dsn")

	if err != nil {
		return err, nil, nil
	}

	return nil, gormDb, &Expecter{adapter: adapter}
}

// NewExpecter returns an Expecter for arbitrary adapters
func NewExpecter(fn AdapterFactory, dialect string, args ...interface{}) (error, *DB, *Expecter) {
	err, gormDb, adapter := fn(dialect, args...)

	if err != nil {
		return err, nil, nil
	}

	return nil, gormDb, &Expecter{adapter: adapter}
}
