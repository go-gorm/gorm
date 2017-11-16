package gorm

import (
	"database/sql"
	"fmt"

	"gopkg.in/DATA-DOG/go-sqlmock.v1"
)

// TestHelper is the exported struct used for setting expectations
type TestHelper struct {
	gormDb   *DB
	mockDb   *sql.DB
	adapter  Adapter
	asserter Asserter
}

// Close closes the DB connection
func (h *TestHelper) Close() error {
	return h.adapter.Close()
}

func (h *TestHelper) ExpectFirst(model interface{}) Query {
	fmt.Printf("Expecting query: %s", "some query\n")
	return h.asserter.Query("some sql")
}

func (h *TestHelper) ExpectFind(model interface{}) {
	fmt.Println("Expecting query: %s", "some query involving Find")
}

type Query interface {
	Return(model interface{}) Query
}

type SqlmockQuery struct {
	query *sqlmock.ExpectedQuery
}

func (q *SqlmockQuery) getRowsForOutType(out interface{}) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{"column1", "column2", "column3"})
	rows = rows.AddRow("someval1", "someval2", "someval3")

	return rows
}

func (q *SqlmockQuery) Return(out interface{}) Query {
	rows := q.getRowsForOutType(out)
	q.query = q.query.WillReturnRows(rows)

	return q
}

type Exec interface {
	Return(args ...interface{})
}

type Adapter interface {
	Open() (error, *sql.DB, *DB, Asserter)
	Close() error
}

type Asserter interface {
	Query(query string) Query
	// Exec(stmt string) Exec
}

type SqlmockAdapter struct {
	mockDb *sql.DB
	mock   *sqlmock.Sqlmock
}

// Open returns the raw sql.DB and a gorm DB instance
func (adapter *SqlmockAdapter) Open() (error, *sql.DB, *DB, Asserter) {
	mockDb, mock, err := sqlmock.NewWithDSN("mock_gorm_dsn")

	if err != nil {
		return err, nil, nil, nil
	}

	gormDb, err := Open("sqlmock", "mock_gorm_dsn")

	if err != nil {
		return err, nil, nil, nil
	}

	return nil, mockDb, gormDb, &SqlmockAsserter{mock: mock, sqlmockDB: mockDb}
}

func (adapter *SqlmockAdapter) Close() error {
	return adapter.mockDb.Close()
}

type SqlmockAsserter struct {
	sqlmockDB *sql.DB
	mock      sqlmock.Sqlmock
}

func (a *SqlmockAsserter) Query(query string) Query {
	q := a.mock.ExpectQuery(query)

	return &SqlmockQuery{q}
}

// NewTestHelper returns a fresh TestHelper
func NewTestHelper(adapter Adapter) (error, *DB, *TestHelper) {
	err, mockDb, gormDb, asserter := adapter.Open()

	if err != nil {
		return err, nil, nil
	}

	return nil, gormDb, &TestHelper{gormDb: gormDb, mockDb: mockDb, adapter: adapter, asserter: asserter}
}
