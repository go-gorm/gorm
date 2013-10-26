package gorm

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
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
	s.Sql = fmt.Sprintf("SELECT * FROM %v %v", s.TableName, s.whereSql())
	return
}

func (s *Orm) query(out interface{}) {
	var (
		is_slice  bool
		dest_type reflect.Type
	)
	dest_out := reflect.Indirect(reflect.ValueOf(out))

	if x := dest_out.Kind(); x == reflect.Slice {
		is_slice = true
		dest_type = dest_out.Type().Elem()
	}

	rows, err := s.db.Query(s.Sql, s.SqlVars...)
	s.Error = err

	for rows.Next() {
		var dest reflect.Value
		if is_slice {
			dest = reflect.New(dest_type).Elem()
		} else {
			dest = reflect.ValueOf(out).Elem()
		}

		columns, _ := rows.Columns()
		var values []interface{}
		for _, value := range columns {
			values = append(values, dest.FieldByName(snakeToUpperCamel(value)).Addr().Interface())
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
	if len(s.whereClause) == 0 {
		return
	} else {
		sql = "WHERE "
		for _, clause := range s.whereClause {
			sql += clause["query"].(string)
			args := clause["args"].([]interface{})
			for _, arg := range args {
				s.SqlVars = append(s.SqlVars, arg.([]interface{})...)
				sql = strings.Replace(sql, "?", "$"+strconv.Itoa(len(s.SqlVars)), 1)
			}
		}
	}
	return
}
