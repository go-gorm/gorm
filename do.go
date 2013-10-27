package gorm

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"

	"strings"
)

type Do struct {
	chain     *Chain
	db        *sql.DB
	driver    string
	TableName string
	Errors    []error

	model     *Model
	value     interface{}
	SqlResult sql.Result

	Sql     string
	SqlVars []interface{}

	whereClause []map[string]interface{}
	orClause    []map[string]interface{}
	selectStr   string
	orderStrs   []string
	offsetStr   string
	limitStr    string
	operation   string
}

func (s *Do) err(err error) {
	if err != nil {
		s.Errors = append(s.Errors, err)
		s.chain.err(err)
	}
}

func (s *Do) setModel(value interface{}) {
	s.value = value
	s.model = &Model{Data: value, driver: s.driver}
	s.TableName = s.model.TableName()
}

func (s *Do) addToVars(value interface{}) string {
	s.SqlVars = append(s.SqlVars, value)
	return fmt.Sprintf("$%d", len(s.SqlVars))
}

func (s *Do) Exec(sql ...string) {
	var err error
	if len(sql) == 0 {
		s.SqlResult, err = s.db.Exec(s.Sql, s.SqlVars...)
	} else {
		s.SqlResult, err = s.db.Exec(sql[0])
	}
	s.err(err)
}

func (s *Do) save() *Do {
	if s.model.PrimaryKeyZero() {
		s.create()
	} else {
		s.update()
	}
	return s
}

func (s *Do) prepareCreateSql() *Do {
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
	return s
}

func (s *Do) create() *Do {
	s.err(s.model.callMethod("BeforeCreate"))
	s.err(s.model.callMethod("BeforeSave"))

	s.prepareCreateSql()

	if len(s.Errors) == 0 {
		var id int64
		if s.driver == "postgres" {
			s.err(s.db.QueryRow(s.Sql, s.SqlVars...).Scan(&id))
		} else {
			var err error
			s.SqlResult, err = s.db.Exec(s.Sql, s.SqlVars...)
			s.err(err)
			id, err = s.SqlResult.LastInsertId()
			s.err(err)
		}
		result := reflect.ValueOf(s.model.Data).Elem()
		result.FieldByName(s.model.PrimaryKey()).SetInt(id)

		s.err(s.model.callMethod("AfterCreate"))
		s.err(s.model.callMethod("AfterSave"))
	}

	return s
}

func (s *Do) prepareUpdateSql() *Do {
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
	return s
}

func (s *Do) update() *Do {
	s.err(s.model.callMethod("BeforeUpdate"))
	s.err(s.model.callMethod("BeforeSave"))
	if len(s.Errors) == 0 {
		s.prepareUpdateSql().Exec()
	}
	s.err(s.model.callMethod("AfterUpdate"))
	s.err(s.model.callMethod("AfterSave"))
	return s
}

func (s *Do) prepareDeleteSql() *Do {
	s.Sql = fmt.Sprintf("DELETE FROM %v %v", s.TableName, s.combinedSql())
	return s
}

func (s *Do) delete() *Do {
	s.err(s.model.callMethod("BeforeDelete"))
	if len(s.Errors) == 0 {
		s.prepareDeleteSql().Exec()
	}
	s.err(s.model.callMethod("AfterDelete"))
	return s
}

func (s *Do) prepareQuerySql() *Do {
	s.Sql = fmt.Sprintf("SELECT %v FROM %v %v", s.selectSql(), s.TableName, s.combinedSql())
	return s
}

func (s *Do) query() {
	var (
		is_slice  bool
		dest_type reflect.Type
	)
	dest_out := reflect.Indirect(reflect.ValueOf(s.value))

	if x := dest_out.Kind(); x == reflect.Slice {
		is_slice = true
		dest_type = dest_out.Type().Elem()
	}

	s.prepareQuerySql()
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
			dest = reflect.ValueOf(s.value).Elem()
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

func (s *Do) pluck(value interface{}) *Do {
	dest_out := reflect.Indirect(reflect.ValueOf(value))
	dest_type := dest_out.Type().Elem()
	s.prepareQuerySql()
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
	return s
}

func (s *Do) buildWhereCondition(clause map[string]interface{}) string {
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

func (s *Do) whereSql() (sql string) {
	var primary_condiation string
	var and_conditions, or_conditions []string

	if !s.model.PrimaryKeyZero() {
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

func (s *Do) selectSql() string {
	if len(s.selectStr) == 0 {
		return " * "
	} else {
		return s.selectStr
	}
}

func (s *Do) orderSql() string {
	if len(s.orderStrs) == 0 {
		return ""
	} else {
		return " ORDER BY " + strings.Join(s.orderStrs, ",")
	}
}

func (s *Do) limitSql() string {
	if len(s.limitStr) == 0 {
		return ""
	} else {
		return " LIMIT " + s.limitStr
	}
}

func (s *Do) offsetSql() string {
	if len(s.offsetStr) == 0 {
		return ""
	} else {
		return " OFFSET " + s.offsetStr
	}
}

func (s *Do) combinedSql() string {
	return s.whereSql() + s.orderSql() + s.limitSql() + s.offsetSql()
}

func (s *Do) createTable() *Do {
	s.Sql = s.model.CreateTable()
	return s
}
