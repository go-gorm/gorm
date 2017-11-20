package gorm

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
)

// Recorder satisfies the logger interface
type Recorder struct {
	stmt string
}

// Print just sets the last recorded SQL statement
// TODO: find a better way to extract SQL from log messages
func (r *Recorder) Print(args ...interface{}) {
	msgs := LogFormatter(args...)
	if len(msgs) >= 4 {
		if v, ok := msgs[3].(string); ok {
			r.stmt = v
		}
	}
}

// AdapterFactory is a generic interface for arbitrary adapters that satisfy
// the interface. variadic args are passed to gorm.Open.
type AdapterFactory func(dialect string, args ...interface{}) (*DB, Adapter, error)

// Expecter is the exported struct used for setting expectations
type Expecter struct {
	// globally scoped expecter
	adapter  Adapter
	noop     SQLCommon
	gorm     *DB
	recorder *Recorder
}

// DefaultNoopDB is a noop db used to get generated sql from gorm.DB
type DefaultNoopDB struct{}

// NoopResult is a noop struct that satisfies sql.Result
type NoopResult struct{}

func (r NoopResult) LastInsertId() (int64, error) {
	return 1, nil
}

func (r NoopResult) RowsAffected() (int64, error) {
	return 1, nil
}

// NewNoopDB initialises a new DefaultNoopDB
func NewNoopDB() SQLCommon {
	return &DefaultNoopDB{}
}

// Exec simulates a sql.DB.Exec
func (r *DefaultNoopDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return NoopResult{}, nil
}

// Prepare simulates a sql.DB.Prepare
func (r *DefaultNoopDB) Prepare(query string) (*sql.Stmt, error) {
	return &sql.Stmt{}, nil
}

// Query simulates a sql.DB.Query
func (r *DefaultNoopDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return nil, errors.New("noop")
}

// QueryRow simulates a sql.DB.QueryRow
func (r *DefaultNoopDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return &sql.Row{}
}

// NewDefaultExpecter returns a Expecter powered by go-sqlmock
func NewDefaultExpecter() (*DB, *Expecter, error) {
	gormDb, adapter, err := NewSqlmockAdapter("sqlmock", "mock_gorm_dsn")

	if err != nil {
		return nil, nil, err
	}

	recorder := &Recorder{}
	noop := &DefaultNoopDB{}
	gorm := &DB{
		db:        noop,
		logger:    recorder,
		logMode:   2,
		values:    map[string]interface{}{},
		callbacks: DefaultCallback,
		dialect:   newDialect("sqlmock", noop),
	}

	gorm.parent = gorm

	return gormDb, &Expecter{adapter: adapter, noop: noop, gorm: gorm, recorder: recorder}, nil
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

// AssertExpectations checks if all expected Querys and Execs were satisfied.
func (h *Expecter) AssertExpectations() error {
	return h.adapter.AssertExpectations()
}

// First triggers a Query
func (h *Expecter) First(out interface{}, where ...interface{}) ExpectedQuery {
	h.gorm.First(out, where...)
	return h.adapter.ExpectQuery(regexp.QuoteMeta(h.recorder.stmt))
}

// Find triggers a Query
func (h *Expecter) Find(out interface{}, where ...interface{}) ExpectedQuery {
	fmt.Printf("Expecting query: %s\n", "some query involving Find")
	return h.adapter.ExpectQuery("some find condition")
}
