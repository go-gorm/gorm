package gorm

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
)

type Recorder struct {
	stmt string
}

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
	noop     NoopDB
	gorm     *DB
	recorder *Recorder
}

type NoopDB interface {
	GetStmts() []string
}

type DefaultNoopDB struct{}

type NoopResult struct{}

func (r NoopResult) LastInsertId() (int64, error) {
	return 1, nil
}

func (r NoopResult) RowsAffected() (int64, error) {
	return 1, nil
}

func NewNoopDB() NoopDB {
	return &DefaultNoopDB{}
}

func (r *DefaultNoopDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return NoopResult{}, nil
}

func (r *DefaultNoopDB) Prepare(query string) (*sql.Stmt, error) {
	return &sql.Stmt{}, nil
}

func (r *DefaultNoopDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return nil, errors.New("noop")
}

func (r *DefaultNoopDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return &sql.Row{}
}

func (r *DefaultNoopDB) GetStmts() []string {
	return []string{"not", "implemented"}
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
