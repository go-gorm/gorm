package gorm

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func (s *Orm) explain(value interface{}, operation string) *Orm {
	s.setModel(value)
	switch operation {
	case "Create":
		s.createSql(value)
	case "Update":
		s.updateSql(value)
	case "Delete":
		s.deleteSql(value)
	case "Query":
		s.querySql(value)
	case "CreateTable":
		s.Sql = s.Model.CreateTable()
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
	defer rows.Close()

	s.Error = err
	if rows.Err() != nil {
		s.Error = rows.Err()
	}

	counts := 0
	for rows.Next() {
		counts += 1
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
	if (counts == 0) && !is_slice {
		s.Error = errors.New("Record not found!")
	}
}

func (s *Orm) createSql(value interface{}) {
	columns, values := s.Model.ColumnsAndValues()
	s.Sql = fmt.Sprintf(
		"INSERT INTO \"%v\" (%v) VALUES (%v) %v",
		s.TableName,
		strings.Join(quoteMap(columns), ","),
		valuesToBinVar(values),
		s.Model.ReturningStr(),
	)
	s.SqlVars = values
	return
}

func (s *Orm) create(value interface{}) {
	var id int64
	if s.driver == "postgres" {
		s.Error = s.db.QueryRow(s.Sql, s.SqlVars...).Scan(&id)
	} else {
		s.SqlResult, s.Error = s.db.Exec(s.Sql, s.SqlVars...)
		id, s.Error = s.SqlResult.LastInsertId()
	}

	result := reflect.ValueOf(s.Model.Data).Elem()
	result.FieldByName(s.Model.PrimaryKey()).SetInt(id)
}

func (s *Orm) updateSql(value interface{}) {
	return
}

func (s *Orm) update(value interface{}) {
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
