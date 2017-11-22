package gorm

import (
	"database/sql/driver"
	"fmt"
	"reflect"

	"github.com/davecgh/go-spew/spew"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

// ExpectedQuery represents an expected query that will be executed and can
// return some rows. It presents a fluent API for chaining calls to other
// expectations
type ExpectedQuery interface {
	Returns(model interface{}) ExpectedQuery
}

// ExpectedExec represents an expected exec that will be executed and can
// return a result. It presents a fluent API for chaining calls to other
// expectations
type ExpectedExec interface {
	Returns(result driver.Result) ExpectedExec
}

// SqlmockQuery implements Query for go-sqlmock
type SqlmockQuery struct {
	queries []*sqlmock.ExpectedQuery
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
			// just append nil if it's not valid, so sqlmock won't complain
			if !value.IsValid() {
				values = append(values, nil)
				continue
			}

			concreteVal := value.Interface()

			if driver.IsValue(concreteVal) {
				values = append(values, concreteVal)
			} else if num, err := driver.DefaultParameterConverter.ConvertValue(concreteVal); err == nil {
				values = append(values, num)
			} else if valuer, ok := concreteVal.(driver.Valuer); ok {
				if convertedValue, err := valuer.Value(); err == nil {
					values = append(values, convertedValue)
				}
			}
		}
	}

	return values
}

func getRelationRows(rVal reflect.Value, fieldName string, relation *Relationship) (*sqlmock.Rows, bool) {
	var (
		rows    *sqlmock.Rows
		columns []string
	)

	switch relation.Kind {
	case "has_one":
		// just a plain struct
		scope := &Scope{Value: rVal.Interface()}

		for _, field := range scope.GetModelStruct().StructFields {
			if field.IsNormal {
				columns = append(columns, field.DBName)
			}
		}

		rows = sqlmock.NewRows(columns)

		// we don't have a slice
		row := getRowForFields(scope.Fields())
		rows = rows.AddRow(row...)

		return rows, true
	case "has_many", "many_to_many":
		// in this case, we're guarnateed to have a slice
		elem := rVal.Type().Elem()
		scope := &Scope{Value: reflect.New(elem).Interface()}

		for _, field := range scope.GetModelStruct().StructFields {
			if field.IsNormal {
				columns = append(columns, field.DBName)
			}
		}

		rows = sqlmock.NewRows(columns)

		// in this case we definitely have a slice
		if rVal.Len() > 0 {
			for i := 0; i < rVal.Len(); i++ {
				scope := &Scope{Value: rVal.Index(i).Interface()}
				row := getRowForFields(scope.Fields())
				rows = rows.AddRow(row...)
			}
		}

		return rows, true
	default:
		return nil, false
	}
}

func (q *SqlmockQuery) getRowsForOutType(out interface{}) []*sqlmock.Rows {
	var (
		columns   []string
		relations = make(map[string]*Relationship)
		rowsSet   []*sqlmock.Rows
	)

	for _, field := range (&Scope{}).New(out).GetModelStruct().StructFields {
		// we get the primary model's columns here
		if field.IsNormal {
			columns = append(columns, field.DBName)
		}

		// check relations
		if !field.IsNormal {
			relationVal := reflect.ValueOf(field.Relationship)
			isNil := relationVal.IsNil()

			if !isNil {
				relations[field.Name] = field.Relationship
			}
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
			rowsSet = append(rowsSet, rows)
		}
	} else if outVal.Kind() == reflect.Struct {
		scope := &Scope{Value: out}
		row := getRowForFields(scope.Fields())
		rows = rows.AddRow(row...)
		rowsSet = append(rowsSet, rows)

		for name, relation := range relations {
			rVal := outVal.FieldByName(name)
			relationRows, hasRows := getRelationRows(rVal, name, relation)

			if hasRows {
				rowsSet = append(rowsSet, relationRows)
			}
		}
	} else {
		panic(fmt.Errorf("Can only get rows for slice or struct"))
	}

	return rowsSet
}

// Returns accepts an out type which should either be a struct or slice. Under
// the hood, it converts a gorm model struct to sql.Rows that can be passed to
// the underlying mock db
func (q *SqlmockQuery) Returns(out interface{}) ExpectedQuery {
	rows := q.getRowsForOutType(out)

	for i, query := range q.queries {
		query.WillReturnRows(rows[i])
		spew.Dump(query)
	}

	return q
}

// SqlmockExec implements Exec for go-sqlmock
type SqlmockExec struct {
	exec *sqlmock.ExpectedExec
}

// Returns accepts a driver.Result. It is passed directly to the underlying
// mock db. Useful for checking DAO behaviour in the event that the incorrect
// number of rows are affected by an Exec
func (e *SqlmockExec) Returns(result driver.Result) ExpectedExec {
	e.exec = e.exec.WillReturnResult(result)

	return e
}
