package gorm

import (
	"database/sql"

	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

type Adapter interface {
	Open() (error, *sql.DB, *DB, Asserter)
	Close() error
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
