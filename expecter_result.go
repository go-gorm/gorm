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
	scope *Scope
	query *sqlmock.ExpectedQuery
}

func (q *SqlmockQuery) getRowsForOutType(out interface{}) *sqlmock.Rows {
	var (
		columns []string
		rows    *sqlmock.Rows
		values  []driver.Value
	)

	q.scope = &Scope{Value: out}
	fields := q.scope.Fields()

	for _, field := range fields {
		if field.IsNormal {
			var (
				column = field.StructField.DBName
				value  = field.Field.Interface()
			)

			if isValue := driver.IsValue(value); isValue {
				columns = append(columns, column)
				values = append(values, value)
			} else if valuer, ok := value.(driver.Valuer); ok {
				if underlyingValue, err := valuer.Value(); err == nil {
					values = append(values, underlyingValue)
					columns = append(columns, field.StructField.DBName)
				}
			}
		}
	}

	rows = sqlmock.NewRows(columns).AddRow(values...)

	return rows
}

func (q *SqlmockQuery) Returns(out interface{}) ExpectedQuery {
	rows := q.getRowsForOutType(out)
	q.query = q.query.WillReturnRows(rows)

	return q
}

type SqlmockExec struct {
	scope *Scope
	exec  *sqlmock.ExpectedExec
}

func (e *SqlmockExec) Returns(result driver.Result) ExpectedExec {
	e.exec = e.exec.WillReturnResult(result)

	return e
}
