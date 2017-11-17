package gorm

import (
	"database/sql"

	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

type Adapter interface {
	Open() (error, Asserter)
	Close() error
}

// SqlmockAdapter implemenets the Adapter interface using go-sqlmock
// it is the default Adapter
type SqlmockAdapter struct {
	mockDb *sql.DB
	mock   *sqlmock.Sqlmock
}

// Open returns the raw sql.DB instance and an Asserter
func (adapter *SqlmockAdapter) Open() (error, Asserter) {
	mockDb, mock, err := sqlmock.NewWithDSN("mock_gorm_dsn")

	adapter.mockDb = mockDb

	if err != nil {
		return err, nil
	}

	return nil, &SqlmockAsserter{mock: mock, sqlmockDB: mockDb}
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
