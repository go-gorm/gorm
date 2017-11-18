package gorm

import (
	"fmt"
)

// AdapterFactory is a generic interface for arbitrary adapters that satisfy
// the interface. variadic args are passed to gorm.Open.
type AdapterFactory func(dialect string, args ...interface{}) (*DB, Adapter, error)

// Expecter is the exported struct used for setting expectations
type Expecter struct {
	Value   interface{}
	adapter Adapter
	search  *search
	values  map[string]interface{}

	// globally scoped expecter
	root *Expecter
}

// NewDefaultExpecter returns a Expecter powered by go-sqlmock
func NewDefaultExpecter() (*DB, *Expecter, error) {
	gormDb, adapter, err := NewSqlmockAdapter("sqlmock", "mock_gorm_dsn")

	if err != nil {
		return nil, nil, err
	}

	return gormDb, &Expecter{adapter: adapter}, nil
}

// NewExpecter returns an Expecter for arbitrary adapters
func NewExpecter(fn AdapterFactory, dialect string, args ...interface{}) (*DB, *Expecter, error) {
	gormDb, adapter, err := fn(dialect, args...)

	if err != nil {
		return nil, nil, err
	}

	return gormDb, &Expecter{adapter: adapter}, nil
}

/* PUBLIC METHODS */

// First triggers a Query
func (h *Expecter) First(model interface{}) ExpectedQuery {
	fmt.Printf("Expecting query: %s", "some query\n")
	return h.adapter.ExpectQuery("some sql")
}

// Find triggers a Query
func (h *Expecter) Find(model interface{}) ExpectedQuery {
	fmt.Printf("Expecting query: %s\n", "some query involving Find")
	return h.adapter.ExpectQuery("some find condition")
}

/* PRIVATE METHODS */

// clone is similar to DB.clone, and ensures that the root Expecter is not
// polluted with subsequent search constraints
func (h *Expecter) clone() *Expecter {
	expecterCopy := &Expecter{
		adapter: h.adapter,
		root:    h.root,
		values:  map[string]interface{}{},
		Value:   h.Value,
	}

	return expecterCopy
}
