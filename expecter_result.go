package gorm

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"regexp"

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
	mock    sqlmock.Sqlmock
	queries []Stmt
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
			// spew.Printf("%v: %v\r\n", field.Name, concreteVal)

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

func (q *SqlmockQuery) getRelationRows(rVal reflect.Value, fieldName string, relation *Relationship) (*sqlmock.Rows, bool) {
	var (
		rows    *sqlmock.Rows
		columns []string
	)

	// we need to check for zero values
	if reflect.DeepEqual(rVal.Interface(), reflect.New(rVal.Type()).Elem().Interface()) {
		// spew.Printf("FOUND EMPTY INTERFACE FOR %s\r\n", fieldName)
		return nil, false
	}

	switch relation.Kind {
	case "has_one":
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
		elem := rVal.Type().Elem()
		scope := &Scope{Value: reflect.New(elem).Interface()}

		for _, field := range scope.GetModelStruct().StructFields {
			if field.IsNormal {
				columns = append(columns, field.DBName)
			}
		}
		// spew.Printf("___GENERATING ROWS FOR %s___\r\n", fieldName)
		// spew.Printf("___COLUMNS___:\r\n%s\r\n", spew.Sdump(columns))
		columns = append(columns, "user_id", "language_id")

		rows = sqlmock.NewRows(columns)

		// in this case we definitely have a slice
		if rVal.Len() > 0 {
			for i := 0; i < rVal.Len(); i++ {
				scope := &Scope{Value: rVal.Index(i).Interface()}
				row := getRowForFields(scope.Fields())
				row = append(row, 1, 1)
				rows = rows.AddRow(row...)
			}

			return rows, true
		}

		return nil, false
	default:
		return nil, false
	}
}

func (q *SqlmockQuery) getDestRows(out interface{}) *sqlmock.Rows {
	var columns []string
	for _, field := range (&Scope{}).New(out).GetModelStruct().StructFields {
		if field.IsNormal {
			columns = append(columns, field.DBName)
		}
	}

	rows := sqlmock.NewRows(columns)
	outVal := indirect(reflect.ValueOf(out))

	// SELECT multiple columns
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
	} else if outVal.Kind() == reflect.Struct { // SELECT with LIMIT 1
		scope := &Scope{Value: out}
		row := getRowForFields(scope.Fields())
		rows = rows.AddRow(row...)
	} else {
		panic(fmt.Errorf("Can only get rows for slice or struct"))
	}

	return rows
}

// Returns accepts an out type which should either be a struct or slice. Under
// the hood, it converts a gorm model struct to sql.Rows that can be passed to
// the underlying mock db
func (q *SqlmockQuery) Returns(out interface{}) ExpectedQuery {
	scope := (&Scope{}).New(out)
	outVal := indirect(reflect.ValueOf(out))

	// rows := q.getRowsForOutType(out)
	destQuery := q.queries[0]
	subQueries := q.queries[1:]

	// main query always at the head of the slice
	q.mock.ExpectQuery(regexp.QuoteMeta(destQuery.sql)).
		WillReturnRows(q.getDestRows(out))

	// subqueries are preload
	for _, subQuery := range subQueries {
		if subQuery.preload != "" {
			if field, ok := scope.FieldByName(subQuery.preload); ok {
				expectation := q.mock.ExpectQuery(regexp.QuoteMeta(subQuery.sql))
				rows, hasRows := q.getRelationRows(outVal.FieldByName(subQuery.preload), subQuery.preload, field.Relationship)

				if hasRows {
					expectation.WillReturnRows(rows)
				}
			}
		}
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
