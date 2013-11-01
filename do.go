package gorm

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"strings"
)

type Do struct {
	chain              *Chain
	db                 *sql.DB
	driver             string
	guessedTableName   string
	specifiedTableName string
	Errors             []error

	model     *Model
	value     interface{}
	sqlResult sql.Result
	sql       string
	sqlVars   []interface{}

	whereClause          []map[string]interface{}
	orClause             []map[string]interface{}
	notClause            []map[string]interface{}
	selectStr            string
	orderStrs            []string
	offsetStr            string
	limitStr             string
	unscoped             bool
	updateAttrs          map[string]interface{}
	ignoreProtectedAttrs bool
}

func (s *Do) tableName() string {
	if s.specifiedTableName == "" {
		var err error
		s.guessedTableName, err = s.model.tableName()
		s.err(err)
		return s.guessedTableName
	} else {
		return s.specifiedTableName
	}
}

func (s *Do) err(err error) error {
	if err != nil {
		s.Errors = append(s.Errors, err)
		s.chain.err(err)
	}
	return err
}

func (s *Do) hasError() bool {
	return len(s.Errors) > 0
}

func (s *Do) setModel(value interface{}) {
	s.model = &Model{data: value, driver: s.driver}
	s.value = value
}

func (s *Do) addToVars(value interface{}) string {
	s.sqlVars = append(s.sqlVars, value)
	return fmt.Sprintf("$%d", len(s.sqlVars))
}

func (s *Do) exec(sql ...string) {
	if s.hasError() {
		return
	}

	var err error
	if len(sql) == 0 {
		if len(s.sql) > 0 {
			s.sqlResult, err = s.db.Exec(s.sql, s.sqlVars...)
		}
	} else {
		s.sqlResult, err = s.db.Exec(sql[0])
	}
	s.err(err)
}

func (s *Do) save() {
	if s.model.primaryKeyZero() {
		s.create()
	} else {
		s.update()
	}
	return
}

func (s *Do) prepareCreateSql() {
	var sqls, columns []string

	for key, value := range s.model.columnsAndValues("create") {
		columns = append(columns, key)
		sqls = append(sqls, s.addToVars(value))
	}

	s.sql = fmt.Sprintf(
		"INSERT INTO \"%v\" (%v) VALUES (%v) %v",
		s.tableName(),
		strings.Join(columns, ","),
		strings.Join(sqls, ","),
		s.model.returningStr(),
	)
	return
}

func (s *Do) create() {
	s.err(s.model.callMethod("BeforeCreate"))
	s.err(s.model.callMethod("BeforeSave"))

	s.prepareCreateSql()

	if !s.hasError() {
		var id int64
		if s.driver == "postgres" {
			s.err(s.db.QueryRow(s.sql, s.sqlVars...).Scan(&id))
		} else {
			var err error
			s.sqlResult, err = s.db.Exec(s.sql, s.sqlVars...)
			s.err(err)
			id, err = s.sqlResult.LastInsertId()
			s.err(err)
		}

		if !s.hasError() {
			result := reflect.ValueOf(s.value).Elem()
			setFieldValue(result.FieldByName(s.model.primaryKey()), id)

			s.err(s.model.callMethod("AfterCreate"))
			s.err(s.model.callMethod("AfterSave"))
		}
	}

	return
}

func (s *Do) setUpdateAttrs(values interface{}, ignore_protected_attrs ...bool) *Do {
	switch values.(type) {
	case map[string]interface{}:
		s.updateAttrs = values.(map[string]interface{})
	case []interface{}:
		for _, value := range values.([]interface{}) {
			s.setUpdateAttrs(value, ignore_protected_attrs...)
		}
	case interface{}:
		m := &Model{data: values, driver: s.driver}
		fields := m.columnsHasValue("")

		s.updateAttrs = make(map[string]interface{}, len(fields))
		for _, field := range fields {
			s.updateAttrs[field.DbName] = field.Value
		}
	}

	s.ignoreProtectedAttrs = len(ignore_protected_attrs) > 0 && ignore_protected_attrs[0]
	return s
}

func (s *Do) prepareUpdateAttrs() (results map[string]interface{}, update bool) {
	if len(s.updateAttrs) > 0 {
		results, update = s.model.updatedColumnsAndValues(s.updateAttrs)
	}
	return
}

func (s *Do) prepareUpdateSql(results map[string]interface{}) {
	var sqls []string
	for key, value := range results {
		sqls = append(sqls, fmt.Sprintf("%v = %v", key, s.addToVars(value)))
	}

	update_attrs := s.model.columnsAndValues("update")
	for key, value := range update_attrs {
		sqls = append(sqls, fmt.Sprintf("%v = %v", key, s.addToVars(value)))
	}

	s.sql = fmt.Sprintf(
		"UPDATE %v SET %v %v",
		s.tableName(),
		strings.Join(sqls, ", "),
		s.combinedSql(),
	)
	return
}

func (s *Do) update() {
	update_attrs := s.updateAttrs
	if len(update_attrs) > 0 {
		var need_update bool
		update_attrs, need_update = s.prepareUpdateAttrs()
		if !need_update {
			return
		}
	}

	s.err(s.model.callMethod("BeforeUpdate"))
	s.err(s.model.callMethod("BeforeSave"))

	s.prepareUpdateSql(update_attrs)

	if !s.hasError() {
		s.exec()

		if !s.hasError() {
			s.err(s.model.callMethod("AfterUpdate"))
			s.err(s.model.callMethod("AfterSave"))
		}
	}
	return
}

func (s *Do) prepareDeleteSql() {
	s.sql = fmt.Sprintf("DELETE FROM %v %v", s.tableName(), s.combinedSql())
	return
}

func (s *Do) delete() {
	s.err(s.model.callMethod("BeforeDelete"))

	if !s.hasError() {
		if !s.unscoped && s.model.hasColumn("DeletedAt") {
			delete_sql := "deleted_at=" + s.addToVars(time.Now())
			s.sql = fmt.Sprintf("UPDATE %v SET %v %v", s.tableName(), delete_sql, s.combinedSql())
			s.exec()
		} else {
			s.prepareDeleteSql()
			s.exec()
		}
		if !s.hasError() {
			s.err(s.model.callMethod("AfterDelete"))
		}
	}
	return
}

func (s *Do) prepareQuerySql() {
	s.sql = fmt.Sprintf("SELECT %v FROM %v %v", s.selectSql(), s.tableName(), s.combinedSql())
	return
}

func (s *Do) query() {
	var (
		is_slice  bool
		dest_type reflect.Type
	)
	dest_out := reflect.Indirect(reflect.ValueOf(s.value))

	if dest_out.Kind() == reflect.Slice {
		is_slice = true
		dest_type = dest_out.Type().Elem()
	}

	s.prepareQuerySql()
	if !s.hasError() {
		rows, err := s.db.Query(s.sql, s.sqlVars...)
		if s.err(err) != nil {
			return
		}

		defer rows.Close()

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
				dest = dest_out
			}

			columns, _ := rows.Columns()
			var values []interface{}
			for _, value := range columns {
				field := dest.FieldByName(snakeToUpperCamel(value))
				if field.IsValid() {
					values = append(values, field.Addr().Interface())
				} else {
					var null interface{}
					values = append(values, &null)
				}
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
}

func (s *Do) count(value interface{}) {
	dest_out := reflect.Indirect(reflect.ValueOf(value))

	s.prepareQuerySql()
	if !s.hasError() {
		rows, err := s.db.Query(s.sql, s.sqlVars...)
		if s.err(err) != nil {
			return
		}

		defer rows.Close()
		for rows.Next() {
			var dest int64
			if s.err(rows.Scan(&dest)) == nil {
				setFieldValue(dest_out, dest)
			}
		}
	}
	return
}

func (s *Do) pluck(column string, value interface{}) {
	s.selectStr = column
	dest_out := reflect.Indirect(reflect.ValueOf(value))

	if dest_out.Kind() != reflect.Slice {
		s.err(errors.New("Return results should be a slice"))
		return
	}
	dest_type := dest_out.Type().Elem()

	s.prepareQuerySql()

	if !s.hasError() {
		rows, err := s.db.Query(s.sql, s.sqlVars...)
		if s.err(err) != nil {
			return
		}

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
	}
	return
}

func (s *Do) where(where ...interface{}) *Do {
	if len(where) > 0 {
		s.whereClause = append(s.whereClause, map[string]interface{}{"query": where[0], "args": where[1:len(where)]})
	}
	return s
}

func (s *Do) primaryCondiation(value interface{}) string {
	return fmt.Sprintf("(%v = %v)", s.model.primaryKeyDb(), value)
}

func (s *Do) buildWhereCondition(clause map[string]interface{}) (str string) {
	query := clause["query"]
	switch query.(type) {
	case string:
		value := query.(string)
		if regexp.MustCompile("^\\s*\\d+\\s*$").MatchString(value) {
			id, _ := strconv.Atoi(value)
			return s.primaryCondiation(s.addToVars(id))
		} else {
			str = "( " + value + " )"
		}
	case int, int64, int32:
		return s.primaryCondiation(s.addToVars(query))
	case []int64, []int, []int32, []string:
		str = fmt.Sprintf("(%v in (?))", s.model.primaryKeyDb())
		clause["args"] = []interface{}{query}
	case map[string]interface{}:
		var sqls []string
		for key, value := range query.(map[string]interface{}) {
			sqls = append(sqls, fmt.Sprintf(" ( %v = %v ) ", key, s.addToVars(value)))
		}
		return strings.Join(sqls, " AND ")
	case interface{}:
		m := &Model{data: query, driver: s.driver}
		var sqls []string
		for _, field := range m.columnsHasValue("") {
			sqls = append(sqls, fmt.Sprintf(" ( %v = %v ) ", field.DbName, s.addToVars(field.Value)))
		}
		return strings.Join(sqls, " AND ")
	}

	args := clause["args"].([]interface{})
	for _, arg := range args {
		switch reflect.TypeOf(arg).Kind() {
		case reflect.Slice: // For where("id in (?)", []int64{1,2})
			values := reflect.ValueOf(arg)
			var temp_marks []string
			for i := 0; i < values.Len(); i++ {
				temp_marks = append(temp_marks, s.addToVars(values.Index(i).Addr().Interface()))
			}
			str = strings.Replace(str, "?", strings.Join(temp_marks, ","), 1)
		default:
			str = strings.Replace(str, "?", s.addToVars(arg), 1)
		}
	}
	return
}

func (s *Do) buildNotCondition(clause map[string]interface{}) (str string) {
	query := clause["query"]
	var not_equal_sql string

	switch query.(type) {
	case string:
		value := query.(string)
		if regexp.MustCompile("^\\s*\\d+\\s*$").MatchString(value) {
			id, _ := strconv.Atoi(value)
			return fmt.Sprintf("(%v <> %v)", s.model.primaryKeyDb(), id)
		} else if regexp.MustCompile("(?i) (=|<>|>|<|LIKE|IS) ").MatchString(value) {
			str = fmt.Sprintf(" NOT (%v) ", value)
			not_equal_sql = fmt.Sprintf(" NOT (%v) ", value)
		} else {
			str = fmt.Sprintf(" (%v NOT IN (?)) ", value)
			not_equal_sql = fmt.Sprintf(" (%v <> ?) ", value)
		}
	case int, int64, int32:
		return fmt.Sprintf("(%v <> %v)", s.model.primaryKeyDb(), query)
	case []int64, []int, []int32, []string:
		if reflect.ValueOf(query).Len() > 0 {
			str = fmt.Sprintf("(%v not in (?))", s.model.primaryKeyDb())
			clause["args"] = []interface{}{query}
		} else {
			return ""
		}
	case map[string]interface{}:
		var sqls []string
		for key, value := range query.(map[string]interface{}) {
			sqls = append(sqls, fmt.Sprintf(" ( %v <> %v ) ", key, s.addToVars(value)))
		}
		return strings.Join(sqls, " AND ")
	case interface{}:
		m := &Model{data: query, driver: s.driver}
		var sqls []string
		for _, field := range m.columnsHasValue("") {
			sqls = append(sqls, fmt.Sprintf(" ( %v <> %v ) ", field.DbName, s.addToVars(field.Value)))
		}
		return strings.Join(sqls, " AND ")
	}

	args := clause["args"].([]interface{})
	for _, arg := range args {
		switch reflect.TypeOf(arg).Kind() {
		case reflect.Slice: // For where("id in (?)", []int64{1,2})
			values := reflect.ValueOf(arg)
			var temp_marks []string
			for i := 0; i < values.Len(); i++ {
				temp_marks = append(temp_marks, s.addToVars(values.Index(i).Addr().Interface()))
			}
			str = strings.Replace(str, "?", strings.Join(temp_marks, ","), 1)
		default:
			str = strings.Replace(not_equal_sql, "?", s.addToVars(arg), 1)
		}
	}
	return
}

func (s *Do) whereSql() (sql string) {
	var primary_condiations, and_conditions, or_conditions []string

	if !s.unscoped && s.model.hasColumn("DeletedAt") {
		primary_condiations = append(primary_condiations, "(deleted_at is null or deleted_at <= '0001-01-02')")
	}

	if !s.model.primaryKeyZero() {
		primary_condiations = append(primary_condiations, s.primaryCondiation(s.addToVars(s.model.primaryKeyValue())))
	}

	for _, clause := range s.whereClause {
		and_conditions = append(and_conditions, s.buildWhereCondition(clause))
	}

	for _, clause := range s.orClause {
		or_conditions = append(or_conditions, s.buildWhereCondition(clause))
	}

	for _, clause := range s.notClause {
		and_conditions = append(and_conditions, s.buildNotCondition(clause))
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

	if len(primary_condiations) > 0 {
		sql = "WHERE " + strings.Join(primary_condiations, " AND ")
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
	var sqls []string
	for _, field := range s.model.fields("") {
		sqls = append(sqls, field.DbName+" "+field.SqlType)
	}

	s.sql = fmt.Sprintf(
		"CREATE TABLE \"%v\" (%v)",
		s.tableName(),
		strings.Join(sqls, ","),
	)
	return s
}

func (s *Do) dropTable() *Do {
	s.sql = fmt.Sprintf(
		"DROP TABLE \"%v\"",
		s.tableName(),
	)
	return s
}

func (s *Do) initializeWithSearchCondition() {
	m := Model{data: s.value, driver: s.driver}

	for _, clause := range s.whereClause {
		query := clause["query"]
		switch query.(type) {
		case map[string]interface{}:
			for key, value := range query.(map[string]interface{}) {
				m.setValueByColumn(key, value, s.value)
			}
		case []interface{}:
			for _, obj := range query.([]interface{}) {
				switch reflect.ValueOf(obj).Kind() {
				case reflect.Struct:
					m := &Model{data: obj, driver: s.driver}
					for _, field := range m.columnsHasValue("") {
						m.setValueByColumn(field.DbName, field.Value, s.value)
					}
				case reflect.Map:
					for key, value := range obj.(map[string]interface{}) {
						m.setValueByColumn(key, value, s.value)
					}
				}
			}
		case interface{}:
			m := &Model{data: query, driver: s.driver}
			for _, field := range m.columnsHasValue("") {
				m.setValueByColumn(field.DbName, field.Value, s.value)
			}
		}
	}
}
