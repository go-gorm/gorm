package gorm

import (
	"errors"
	"fmt"
	"reflect"
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

		if is_slice {
			dest_out.Set(reflect.Append(dest_out, dest))
		}
	}

	if (counts == 0) && !is_slice {
		s.Error = errors.New("Record not found!")
	}
}

func (s *Orm) createSql(value interface{}) {
	columns, values := s.Model.ColumnsAndValues()

	var sqls []string
	for _, value := range values {
		sqls = append(sqls, s.addToVars(value))
	}

	s.Sql = fmt.Sprintf(
		"INSERT INTO \"%v\" (%v) VALUES (%v) %v",
		s.TableName,
		strings.Join(s.quoteMap(columns), ","),
		strings.Join(sqls, ","),
		s.Model.ReturningStr(),
	)
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
	columns, values := s.Model.ColumnsAndValues()
	var sets []string
	for index, column := range columns {
		sets = append(sets, fmt.Sprintf("%v = %v", s.quote(column), s.addToVars(values[index])))
	}

	s.Sql = fmt.Sprintf(
		"UPDATE %v SET %v %v",
		s.TableName,
		strings.Join(sets, ", "),
		s.whereSql(),
	)

	return
}

func (s *Orm) update(value interface{}) {
	s.Exec()
	return
}

func (s *Orm) deleteSql(value interface{}) {
	s.Sql = fmt.Sprintf("DELETE FROM %v %v", s.TableName, s.whereSql())
	return
}

func (s *Orm) whereSql() (sql string) {
	var conditions []string
	if !s.Model.PrimaryKeyIsEmpty() {
		conditions = append(conditions, fmt.Sprintf("(%v = %v)", s.quote(s.Model.PrimaryKeyDb()), s.addToVars(s.Model.PrimaryKeyValue())))
	}

	if len(s.whereClause) > 0 {
		for _, clause := range s.whereClause {
			str := "( " + clause["query"].(string) + " )"
			args := clause["args"].([]interface{})
			for _, arg := range args {
				switch reflect.TypeOf(arg).Kind() {
				case reflect.Slice: // For where("id in (?)", []int64{1,2})
					v := reflect.ValueOf(arg)

					var temp_marks []string
					for i := 0; i < v.Len(); i++ {
						temp_marks = append(temp_marks, "?")
					}

					str = strings.Replace(str, "?", strings.Join(temp_marks, ","), 1)

					for i := 0; i < v.Len(); i++ {
						str = strings.Replace(str, "?", s.addToVars(v.Index(i).Addr().Interface()), 1)
					}
				default:
					str = strings.Replace(str, "?", s.addToVars(arg), 1)
				}
			}
			conditions = append(conditions, str)
		}
	}

	if len(conditions) > 0 {
		sql = "WHERE " + strings.Join(conditions, " AND ")
	}
	return
}

func (s *Orm) addToVars(value interface{}) string {
	s.SqlVars = append(s.SqlVars, value)
	return fmt.Sprintf("$%d", len(s.SqlVars))
}
