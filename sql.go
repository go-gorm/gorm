package gorm

import (
	"reflect"
	"strings"

	"fmt"
)

func (s *Orm) explain(value interface{}, operation string) *Orm {
	s.setModel(value)
	switch operation {
	case "Save":
		s.saveSql(value)
	case "Delete":
		s.deleteSql(value)
	case "Query":
		s.querySql(value)
	}
	return s
}

func (s *Orm) querySql(out interface{}) {
	s.Sql = "SELECT * from users limit 1"
	return
}

func (s *Orm) query(out interface{}) {
	rows, err := s.db.Query(s.Sql)
	s.Error = err
	for rows.Next() {
		dest := reflect.ValueOf(out).Elem()
		fmt.Printf("%+v", dest)
		columns, _ := rows.Columns()
		var values []interface{}
		for _, value := range columns {
			values = append(values, dest.FieldByName(value).Addr().Interface())
		}
		s.Error = rows.Scan(values...)
	}
	return
}

func (s *Orm) saveSql(value interface{}) {
	columns, values := modelValues(value)
	s.Sql = fmt.Sprintf(
		"INSERT INTO \"%v\" (%v) VALUES (%v)",
		s.TableName,
		strings.Join(quoteMap(columns), ","),
		valuesToBinVar(values),
	)
	s.SqlVars = values
	return
}

func (s *Orm) deleteSql(value interface{}) {
	s.Sql = fmt.Sprintf("DELETE FROM %v WHERE %v", s.TableName, s.whereSql)
	return
}

func (s *Orm) whereSql() (sql string) {
	sql = "1=1"
	return
}
