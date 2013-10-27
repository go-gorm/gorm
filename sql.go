package gorm

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

func (s *Orm) validSql(str string) (result bool) {
	result = regexp.MustCompile("^\\s*[\\w][\\w\\s,.]*[\\w]\\s*$").MatchString(str)
	if !result {
		s.err(errors.New(fmt.Sprintf("SQL is not valid, %s", str)))
	}
	return
}

func (s *Orm) explain(value interface{}, operation string) *Orm {
	s.Model(value)

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
		s.Sql = s.model.CreateTable()
	}
	return s
}

func (s *Orm) querySql(out interface{}) {
	s.Sql = fmt.Sprintf("SELECT %v FROM %v %v", s.selectSql(), s.TableName, s.combinedSql())
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
	s.err(err)
	if rows.Err() != nil {
		s.err(rows.Err())
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
		s.err(rows.Scan(values...))

		if is_slice {
			dest_out.Set(reflect.Append(dest_out, dest))
		}
	}

	if (counts == 0) && !is_slice {
		s.err(errors.New("Record not found!"))
	}
}

func (s *Orm) pluck(value interface{}) {
	dest_out := reflect.Indirect(reflect.ValueOf(value))
	dest_type := dest_out.Type().Elem()

	rows, err := s.db.Query(s.Sql, s.SqlVars...)
	s.err(err)

	defer rows.Close()
	for rows.Next() {
		dest := reflect.New(dest_type).Elem().Interface()
		s.err(rows.Scan(&dest))
		switch dest.(type) {
		case []uint8:
			if dest_type.String() == "string" {
				dest = string(dest.([]uint8))
			}
			dest_out.Set(reflect.Append(dest_out, reflect.ValueOf(dest)))
		default:
			dest_out.Set(reflect.Append(dest_out, reflect.ValueOf(dest)))
		}
	}
	return
}

func (s *Orm) createSql(value interface{}) {
	columns, values := s.model.ColumnsAndValues("create")

	var sqls []string
	for _, value := range values {
		sqls = append(sqls, s.addToVars(value))
	}

	s.Sql = fmt.Sprintf(
		"INSERT INTO \"%v\" (%v) VALUES (%v) %v",
		s.TableName,
		strings.Join(s.quoteMap(columns), ","),
		strings.Join(sqls, ","),
		s.model.ReturningStr(),
	)
	return
}

func (s *Orm) create(value interface{}) {
	var id int64
	s.err(s.model.callMethod("BeforeCreate"))
	s.err(s.model.callMethod("BeforeSave"))

	if s.driver == "postgres" {
		s.err(s.db.QueryRow(s.Sql, s.SqlVars...).Scan(&id))
	} else {
		var err error
		s.SqlResult, err = s.db.Exec(s.Sql, s.SqlVars...)
		s.err(err)
		id, err = s.SqlResult.LastInsertId()
		s.err(err)
	}

	s.err(s.model.callMethod("AfterCreate"))
	s.err(s.model.callMethod("AfterSave"))

	result := reflect.ValueOf(s.model.Data).Elem()
	result.FieldByName(s.model.PrimaryKey()).SetInt(id)
}

func (s *Orm) updateSql(value interface{}) {
	columns, values := s.model.ColumnsAndValues("update")
	var sets []string
	for index, column := range columns {
		sets = append(sets, fmt.Sprintf("%v = %v", s.quote(column), s.addToVars(values[index])))
	}

	s.Sql = fmt.Sprintf(
		"UPDATE %v SET %v %v",
		s.TableName,
		strings.Join(sets, ", "),
		s.combinedSql(),
	)

	return
}

func (s *Orm) update(value interface{}) {
	s.err(s.model.callMethod("BeforeUpdate"))
	s.err(s.model.callMethod("BeforeSave"))
	s.Exec()
	s.err(s.model.callMethod("AfterUpdate"))
	s.err(s.model.callMethod("AfterSave"))
	return
}

func (s *Orm) deleteSql(value interface{}) {
	s.Sql = fmt.Sprintf("DELETE FROM %v %v", s.TableName, s.combinedSql())
	return
}

func (s *Orm) delete(value interface{}) {
	s.err(s.model.callMethod("BeforeDelete"))
	s.Exec()
	s.err(s.model.callMethod("AfterDelete"))
}

func (s *Orm) buildWhereCondition(clause map[string]interface{}) string {
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
	return str
}

func (s *Orm) whereSql() (sql string) {
	var primary_condiation string
	var and_conditions, or_conditions []string

	if !s.model.PrimaryKeyIsEmpty() {
		primary_condiation = fmt.Sprintf("(%v = %v)", s.quote(s.model.PrimaryKeyDb()), s.addToVars(s.model.PrimaryKeyValue()))
	}

	for _, clause := range s.whereClause {
		and_conditions = append(and_conditions, s.buildWhereCondition(clause))
	}

	for _, clause := range s.orClause {
		or_conditions = append(or_conditions, s.buildWhereCondition(clause))
	}

	and_sql := strings.Join(and_conditions, " AND ")
	or_sql := strings.Join(or_conditions, " OR ")
	combined_conditions := and_sql
	if len(combined_conditions) > 0 {
		if len(or_sql) > 0 {
			combined_conditions = combined_conditions + " OR " + or_sql
		}
	} else {
		combined_conditions = or_sql
	}

	if len(primary_condiation) > 0 {
		sql = "WHERE " + primary_condiation
		if len(combined_conditions) > 0 {
			sql = sql + " AND ( " + combined_conditions + " )"
		}
	} else if len(combined_conditions) > 0 {
		sql = "WHERE " + combined_conditions
	}
	return
}

func (s *Orm) selectSql() string {
	if len(s.selectStr) == 0 {
		return " * "
	} else {
		return s.selectStr
	}
}

func (s *Orm) orderSql() string {
	if len(s.orderStrs) == 0 {
		return ""
	} else {
		return " ORDER BY " + strings.Join(s.orderStrs, ",")
	}
}

func (s *Orm) limitSql() string {
	if len(s.limitStr) == 0 {
		return ""
	} else {
		return " LIMIT " + s.limitStr
	}
}

func (s *Orm) offsetSql() string {
	if len(s.offsetStr) == 0 {
		return ""
	} else {
		return " OFFSET " + s.offsetStr
	}
}

func (s *Orm) combinedSql() string {
	return s.whereSql() + s.orderSql() + s.limitSql() + s.offsetSql()
}

func (s *Orm) addToVars(value interface{}) string {
	s.SqlVars = append(s.SqlVars, value)
	return fmt.Sprintf("$%d", len(s.SqlVars))
}
