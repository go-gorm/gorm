package gorm

import (
	"database/sql/driver"
	"fmt"
	"reflect"

	"github.com/davecgh/go-spew/spew"
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

func getRowForFields(fields []*Field) []driver.Value {
	var values []driver.Value
	for _, field := range fields {
		if field.IsNormal {
			value := field.Field

			// dereference pointers
			if field.Field.Kind() == reflect.Ptr {
				value = reflect.Indirect(field.Field)
			}

			// check if we have a zero Value
			if !value.IsValid() {
				values = append(values, nil)
				continue
			}

			concreteVal := value.Interface()

			// if we already have a driver.Value, just append
			_, isValuer := concreteVal.(driver.Valuer)
			spew.Printf("%s: %v\r\n", field.DBName, isValuer)

			if driver.IsValue(concreteVal) {
				values = append(values, concreteVal)
			} else if valuer, ok := concreteVal.(driver.Valuer); ok {
				if convertedValue, err := valuer.Value(); err == nil {
					values = append(values, convertedValue)
				}
			}
		}
	}

	return values
}

func (q *SqlmockQuery) getRowsForOutType(out interface{}) *sqlmock.Rows {
	var columns []string

	for _, field := range (&Scope{}).New(out).GetModelStruct().StructFields {
		if field.IsNormal {
			columns = append(columns, field.DBName)
		}
	}

	rows := sqlmock.NewRows(columns)

	outVal := indirect(reflect.ValueOf(out))

	if outVal.Kind() == reflect.Slice {
		outSlice := []interface{}{}
		for i := 0; i < outVal.Len(); i++ {
			outSlice = append(outSlice, outVal.Index(i).Interface())
		}

		for _, outElem := range outSlice {
			scope := &Scope{Value: outElem}
			row := getRowForFields(scope.Fields())
			rows = rows.AddRow(row...)
		}
	} else if outVal.Kind() == reflect.Struct {
		scope := &Scope{Value: out}
		row := getRowForFields(scope.Fields())
		rows = rows.AddRow(row...)
	} else {
		panic(fmt.Errorf("Can only get rows for slice or struct"))
	}

	spew.Dump(columns)
	spew.Dump(rows)

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
