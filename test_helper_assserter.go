package gorm

import sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"

type Query interface {
	Return(model interface{}) Query
}

type Exec interface {
	Return(args ...interface{}) Exec
}

type Asserter interface {
	Query(query string) Query
	// Exec(stmt string) Exec
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

func (q *SqlmockQuery) Return(out interface{}) Query {
	rows := q.getRowsForOutType(out)
	q.query = q.query.WillReturnRows(rows)

	return q
}
