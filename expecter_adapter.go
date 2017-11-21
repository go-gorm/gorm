package gorm

import (
	"database/sql"

	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

var (
	db   *sql.DB
	mock sqlmock.Sqlmock
)

func init() {
	var err error

	db, mock, err = sqlmock.NewWithDSN("mock_gorm_dsn")

	if err != nil {
		panic(err.Error())
	}
}

// Adapter provides an abstract interface over concrete mock database
// implementations (e.g. go-sqlmock or go-testdb)
type Adapter interface {
	ExpectQuery(stmts ...string) ExpectedQuery
	ExpectExec(stmt string) ExpectedExec
	AssertExpectations() error
}

// SqlmockAdapter implemenets the Adapter interface using go-sqlmock
// it is the default Adapter
type SqlmockAdapter struct {
	db     *sql.DB
	mocker sqlmock.Sqlmock
}

// NewSqlmockAdapter returns a mock gorm.DB and an Adapter backed by
// go-sqlmock
func NewSqlmockAdapter(dialect string, args ...interface{}) (*DB, Adapter, error) {
	gormDb, err := Open("sqlmock", "mock_gorm_dsn")

	if err != nil {
		return nil, nil, err
	}

	return gormDb, &SqlmockAdapter{db: db, mocker: mock}, nil
}

// ExpectQuery wraps the underlying mock method for setting a query
// expectation. It accepts multiple statements in the event of preloading
func (a *SqlmockAdapter) ExpectQuery(stmts ...string) ExpectedQuery {
	var queries []*sqlmock.ExpectedQuery

	for _, stmt := range stmts {
		queries = append(queries, a.mocker.ExpectQuery(stmt))
	}

	return &SqlmockQuery{queries: queries}
}

// ExpectExec wraps the underlying mock method for setting a exec
// expectation
func (a *SqlmockAdapter) ExpectExec(stmt string) ExpectedExec {
	e := a.mocker.ExpectExec(stmt)

	return &SqlmockExec{exec: e}
}

// AssertExpectations asserts that _all_ expectations for a test have been met
// and returns an error specifying which have not if there are unmet
// expectations
func (a *SqlmockAdapter) AssertExpectations() error {
	return a.mocker.ExpectationsWereMet()
}
