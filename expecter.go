package gorm

import (
	"fmt"
)

// Recorder satisfies the logger interface
type Recorder struct {
	stmts   []Stmt
	preload []searchPreload // store it on Recorder
}

// Stmt represents a sql statement. It can be an Exec, Query, or QueryRow
type Stmt struct {
	kind    string // can be Query, Exec, QueryRow
	preload string // contains schema if it is a preload query
	sql     string
	args    []interface{}
}

func recordExecCallback(scope *Scope) {
	r, ok := scope.Get("gorm:recorder")

	if !ok {
		panic(fmt.Errorf("Expected a recorder to be set, but got none"))
	}

	stmt := Stmt{
		kind: "exec",
		sql:  scope.SQL,
		args: scope.SQLVars,
	}

	recorder := r.(*Recorder)

	recorder.Record(stmt)
}

func recordQueryCallback(scope *Scope) {
	r, ok := scope.Get("gorm:recorder")

	if !ok {
		panic(fmt.Errorf("Expected a recorder to be set, but got none"))
	}

	recorder := r.(*Recorder)

	stmt := Stmt{
		kind: "query",
		sql:  scope.SQL,
		args: scope.SQLVars,
	}

	if len(recorder.preload) > 0 {
		// this will cause the scope.SQL to mutate to the preload query
		stmt.preload = recorder.preload[0].schema

		// we just want to pop the first element off
		recorder.preload = recorder.preload[1:]
	}

	recorder.Record(stmt)
}

func recordPreloadCallback(scope *Scope) {
	// this callback runs _before_ gorm:preload
	// it should record the next thing to be preloaded
	recorder, ok := scope.Get("gorm:recorder")

	if !ok {
		panic(fmt.Errorf("Expected a recorder to be set, but got none"))
	}

	if len(scope.Search.preload) > 0 {
		// spew.Printf("callback:preload\r\n%s\r\n", spew.Sdump(scope.Search.preload))
		recorder.(*Recorder).preload = scope.Search.preload
	}
}

// Record records a Stmt for use when SQL is finally executed
func (r *Recorder) Record(stmt Stmt) {
	r.stmts = append(r.stmts, stmt)
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
		logger:    defaultLogger,
		values:    map[string]interface{}{},
		callbacks: DefaultCallback,
		dialect:   newDialect("sqlmock", noop),
	}

	gorm.parent = gorm
	gorm = gorm.Set("gorm:recorder", recorder)
	gorm.Callback().Create().After("gorm:create").Register("gorm:record_exec", recordExecCallback)
	gorm.Callback().Query().Before("gorm:preload").Register("gorm:record_preload", recordPreloadCallback)
	gorm.Callback().Query().After("gorm:query").Register("gorm:record_query", recordQueryCallback)
	gorm.Callback().RowQuery().After("gorm:row_query").Register("gorm:record_query", recordQueryCallback)
	gorm.Callback().Update().After("gorm:update").Register("gorm:record_exec", recordExecCallback)

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

// Model sets scope.Value
func (h *Expecter) Model(model interface{}) *Expecter {
	h.gorm = h.gorm.Model(model)
	return h
}

/* CREATE */

// Create mocks insertion of a model into the DB
func (h *Expecter) Create(model interface{}) ExpectedExec {
	h.gorm.Create(model)
	return h.adapter.ExpectExec(h.recorder.stmts[0])
}

/* READ */

// First triggers a Query
func (h *Expecter) First(out interface{}, where ...interface{}) ExpectedQuery {
	h.gorm.First(out, where...)
	return h.adapter.ExpectQuery(h.recorder.stmts...)
}

// Find triggers a Query
func (h *Expecter) Find(out interface{}, where ...interface{}) ExpectedQuery {
	h.gorm.Find(out, where...)
	return h.adapter.ExpectQuery(h.recorder.stmts...)
}

// Preload clones the expecter and sets a preload condition on gorm.DB
func (h *Expecter) Preload(column string, conditions ...interface{}) *Expecter {
	clone := h.clone()
	clone.gorm = clone.gorm.Preload(column, conditions...)

	return clone
}

/* UPDATE */

// Save mocks updating a record in the DB and will trigger db.Exec()
func (h *Expecter) Save(model interface{}) ExpectedExec {
	h.gorm.Save(model)
	return h.adapter.ExpectExec(h.recorder.stmts[0])
}

// Update mocks updating the given attributes in the DB
func (h *Expecter) Update(attrs ...interface{}) ExpectedExec {
	h.gorm.Update(attrs...)
	return h.adapter.ExpectExec(h.recorder.stmts[0])
}

// Updates does the same thing as Update, but with map or struct
func (h *Expecter) Updates(values interface{}, ignoreProtectedAttrs ...bool) ExpectedExec {
	h.gorm.Updates(values, ignoreProtectedAttrs...)
	return h.adapter.ExpectExec(h.recorder.stmts[0])
}

/* PRIVATE METHODS */

func (h *Expecter) clone() *Expecter {
	return &Expecter{
		adapter:  h.adapter,
		gorm:     h.gorm,
		recorder: h.recorder,
	}
}
