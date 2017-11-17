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
	ExpectQuery(stmt string) ExpectedQuery
	ExpectExec(stmt string) ExpectedExec
}

// SqlmockAdapter implemenets the Adapter interface using go-sqlmock
// it is the default Adapter
type SqlmockAdapter struct {
	db     *sql.DB
	mocker sqlmock.Sqlmock
}

func NewSqlmockAdapter(dialect string, args ...interface{}) (error, *DB, Adapter) {
	gormDb, err := Open("sqlmock", "mock_gorm_dsn")

	if err != nil {
		return err, nil, nil
	}

	return nil, gormDb, &SqlmockAdapter{db: db, mocker: mock}
}

func (a *SqlmockAdapter) ExpectQuery(stmt string) ExpectedQuery {
	q := a.mocker.ExpectQuery(stmt)

	return &SqlmockQuery{query: q}
}

func (a *SqlmockAdapter) ExpectExec(stmt string) ExpectedExec {
	e := a.mocker.ExpectExec(stmt)

	return &SqlmockExec{exec: e}
}
