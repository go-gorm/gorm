package gorm

import (
	"database/sql/driver"

	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

type ExpectedQuery interface {
	Returns(model interface{}) ExpectedQuery
}

type ExpectedExec interface {
	Returns(result driver.Result) ExpectedExec
}

// SqlmockQuery implements Query for asserter go-sqlmock
type SqlmockQuery struct {
	query *sqlmock.ExpectedQuery
}

func (q *SqlmockQuery) getRowsForOutType(out interface{}) *sqlmock.Rows {
	rows := sqlmock.NewRows([]string{"column1", "column2", "column3"})
	rows = rows.AddRow("someval1", "someval2", "someval3")

	return rows
}

func (q *SqlmockQuery) Returns(out interface{}) ExpectedQuery {
	rows := q.getRowsForOutType(out)
	q.query = q.query.WillReturnRows(rows)

	return q
}

type SqlmockExec struct {
	exec *sqlmock.ExpectedExec
}

func (e *SqlmockExec) Returns(result driver.Result) ExpectedExec {
	e.exec = e.exec.WillReturnResult(result)

	return e
}
