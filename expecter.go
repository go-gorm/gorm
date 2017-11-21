package gorm

import (
	"regexp"
)

// Recorder satisfies the logger interface
type Recorder struct {
	stmts []Stmt
}

// Stmt represents a sql statement. It can be an Exec or Query
type Stmt struct {
	stmtType string
	sql      string
	args     []interface{}
}

func getStmtFromLog(values ...interface{}) Stmt {
	var statement Stmt

	if len(values) > 1 {
		var (
			level = values[0]
		)

		if level == "sql" {
			statement.args = values[4].([]interface{})
			statement.sql = values[3].(string)
		}

		return statement
	}

	return statement
}

// Print just sets the last recorded SQL statement
// TODO: find a better way to extract SQL from log messages
func (r *Recorder) Print(args ...interface{}) {
	statement := getStmtFromLog(args...)

	if statement.sql != "" {
		r.stmts = append(r.stmts, statement)
	}
}

// GetFirst returns the first recorded sql statement logged. If there are no
// statements, false is returned
func (r *Recorder) GetFirst() (Stmt, bool) {
	var stmt Stmt
	if len(r.stmts) > 0 {
		stmt = r.stmts[0]
		return stmt, true
	}

	return stmt, false
}

// IsEmpty returns true if the statements slice is empty
func (r *Recorder) IsEmpty() bool {
	return len(r.stmts) == 0
}

// AdapterFactory is a generic interface for arbitrary adapters that satisfy
// the interface. variadic args are passed to gorm.Open.
type AdapterFactory func(dialect string, args ...interface{}) (*DB, Adapter, error)

// Expecter is the exported struct used for setting expectations
type Expecter struct {
	// globally scoped expecter
	adapter  Adapter
	gorm     *DB
	recorder *Recorder
}

// NewDefaultExpecter returns a Expecter powered by go-sqlmock
func NewDefaultExpecter() (*DB, *Expecter, error) {
	gormDb, adapter, err := NewSqlmockAdapter("sqlmock", "mock_gorm_dsn")

	if err != nil {
		return nil, nil, err
	}

	recorder := &Recorder{}
	noop, _ := NewNoopDB()
	gorm := &DB{
		db:        noop,
		logger:    recorder,
		logMode:   2,
		values:    map[string]interface{}{},
		callbacks: DefaultCallback,
		dialect:   newDialect("sqlmock", noop),
	}

	gorm.parent = gorm

	return gormDb, &Expecter{adapter: adapter, gorm: gorm, recorder: recorder}, nil
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
	var q ExpectedQuery
	h.gorm.First(out, where...)

	if empty := h.recorder.IsEmpty(); empty {
		panic("No recorded statements")
	}

	for _, stmt := range h.recorder.stmts {
		q = h.adapter.ExpectQuery(regexp.QuoteMeta(stmt.sql))
	}

	return q
}

// Find triggers a Query
func (h *Expecter) Find(out interface{}, where ...interface{}) ExpectedQuery {
	var q ExpectedQuery
	h.gorm.Find(out, where...)

	if empty := h.recorder.IsEmpty(); empty {
		panic("No recorded statements")
	}

	for _, stmt := range h.recorder.stmts {
		q = h.adapter.ExpectQuery(regexp.QuoteMeta(stmt.sql))
	}

	return q
}

// Preload clones the expecter and sets a preload condition on gorm.DB
func (h *Expecter) Preload(column string, conditions ...interface{}) *Expecter {
	clone := h.clone()
	clone.gorm = clone.gorm.Preload(column, conditions...)

	return clone
}

/* PRIVATE METHODS */

func (h *Expecter) clone() *Expecter {
	return &Expecter{
		adapter:  h.adapter,
		gorm:     h.gorm,
		recorder: h.recorder,
	}
}
