package gorm

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Do struct {
	chain              *Chain
	db                 sql_common
	driver             string
	guessedTableName   string
	specifiedTableName string

	model   *Model
	value   interface{}
	sql     string
	sqlVars []interface{}

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
	if len(s.specifiedTableName) == 0 {
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
		s.chain.err(err)
	}
	return err
}

func (s *Do) hasError() bool {
	return s.chain.hasError()
}

func (s *Do) setModel(value interface{}) *Do {
	s.model = &Model{data: value, driver: s.driver}
	s.value = value
	return s
}

func (s *Do) addToVars(value interface{}) string {
	s.sqlVars = append(s.sqlVars, value)
	if s.driver == "postgres" {
		return fmt.Sprintf("$%d", len(s.sqlVars))
	} else {
		return "?"
	}
}

func (s *Do) exec(sqls ...string) (err error) {
	if s.hasError() {
		return
	} else {
		if len(sqls) > 0 {
			s.sql = sqls[0]
		}
		_, err = s.db.Exec(s.sql, s.sqlVars...)
		slog(s.sql, s.sqlVars...)
	}
	return s.err(err)
}

func (s *Do) save() (i interface{}) {
	if s.model.primaryKeyZero() {
		return s.create()
	} else {
		return s.update()
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
		"INSERT INTO %v (%v) VALUES (%v) %v",
		s.tableName(),
		strings.Join(columns, ","),
		strings.Join(sqls, ","),
		s.model.returningStr(),
	)
	return
}

func (s *Do) saveBeforeAssociations() {
	for _, field := range s.model.beforeAssociations() {
		var id interface{}
		do := &Do{chain: s.chain, db: s.db, driver: s.driver}

		reflect_value := reflect.ValueOf(field.Value)
		if reflect_value.CanAddr() {
			id = do.setModel(reflect_value.Addr().Interface()).save()
		} else {
			dest_value := reflect.New(reflect_value.Type()).Elem()
			m := &Model{data: field.Value, driver: s.driver}
			for _, f := range m.columnsHasValue("other") {
				dest_value.FieldByName(f.Name).Set(reflect.ValueOf(f.Value))
			}
			id = do.setModel(dest_value.Addr().Interface()).save()
			m.setValueByColumn(field.Name, dest_value.Interface(), s.value)
		}

		if len(field.foreignKey) > 0 {
			s.model.setValueByColumn(field.foreignKey, id, s.model.data)
		}
	}
}

func (s *Do) saveAfterAssociations() {
	for _, field := range s.model.afterAssociations() {
		reflect_value := reflect.ValueOf(field.Value)

		switch reflect.TypeOf(field.Value).Kind() {
		case reflect.Slice:
			for i := 0; i < reflect_value.Len(); i++ {
				value := reflect_value.Index(i).Addr().Interface()
				do := &Do{chain: s.chain, db: s.db, driver: s.driver}
				if len(field.foreignKey) > 0 {
					s.model.setValueByColumn(field.foreignKey, s.model.primaryKeyValue(), value)
				}
				do.setModel(value).save()
			}
		default:
			do := &Do{chain: s.chain, db: s.db, driver: s.driver}
			if reflect_value.CanAddr() {
				s.model.setValueByColumn(field.foreignKey, s.model.primaryKeyValue(), field.Value)
				do.setModel(field.Value).save()
			} else {
				dest_value := reflect.New(reflect.TypeOf(field.Value)).Elem()
				m := &Model{data: field.Value, driver: s.driver}
				for _, f := range m.columnsHasValue("other") {
					dest_value.FieldByName(f.Name).Set(reflect.ValueOf(f.Value))
				}

				setFieldValue(dest_value.FieldByName(field.foreignKey), s.model.primaryKeyValue())
				do.setModel(dest_value.Addr().Interface()).save()

				m.setValueByColumn(field.Name, dest_value.Interface(), s.value)
			}
		}
	}
}

func (s *Do) create() (i interface{}) {
	s.err(s.model.callMethod("BeforeCreate"))
	s.err(s.model.callMethod("BeforeSave"))

	s.saveBeforeAssociations()
	s.prepareCreateSql()

	if !s.hasError() {
		var id interface{}
		if s.driver == "postgres" {
			s.err(s.db.QueryRow(s.sql, s.sqlVars...).Scan(&id))
		} else {
			if sql_result, err := s.db.Exec(s.sql, s.sqlVars...); s.err(err) == nil {
				id, err = sql_result.LastInsertId()
				s.err(err)
			}
		}
		slog(s.sql, s.sqlVars...)

		if !s.hasError() {
			result := reflect.Indirect(reflect.ValueOf(s.value))
			if !setFieldValue(result.FieldByName(s.model.primaryKey()), id) {
				fmt.Printf("Can't set primary key for %#v\n", result.Interface())
			}
			s.saveAfterAssociations()

			s.err(s.model.callMethod("AfterCreate"))
			s.err(s.model.callMethod("AfterSave"))
		}
		return id
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
		fields := m.columnsHasValue("other")

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

func (s *Do) update() (i int64) {
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

	s.saveBeforeAssociations()
	s.prepareUpdateSql(update_attrs)

	if !s.hasError() {
		s.exec()
		s.saveAfterAssociations()

		if !s.hasError() {
			s.err(s.model.callMethod("AfterUpdate"))
			s.err(s.model.callMethod("AfterSave"))
		}
	}

	return s.model.primaryKeyValue()
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

func (s *Do) first() {
	s.limitStr = "1"
	s.orderStrs = append(s.orderStrs, s.model.primaryKeyDb())
	s.query()
}

func (s *Do) last() {
	s.limitStr = "1"
	s.orderStrs = append(s.orderStrs, s.model.primaryKeyDb()+" DESC")
	s.query()
}

func (s *Do) getForeignKey(from *Model, to *Model, foreign_key string) (err error, from_from bool, foreign_value interface{}) {
	if has_column, is_slice, value := from.ColumnAndValue(foreign_key); has_column {
		from_from = true
		if is_slice {
			foreign_value = to.primaryKeyValue()
		} else {
			foreign_value = value
		}
	} else if has_column, _, _ := to.ColumnAndValue(foreign_key); has_column {
		foreign_value = from.primaryKeyValue()
	} else {
		err = errors.New("Can't find valid foreign Key")
	}
	return
}

func (s *Do) related(value interface{}, foreign_keys ...string) {
	var foreign_value interface{}
	var from_from bool
	var foreign_key string
	var err error

	from := &Model{data: value, driver: s.driver}
	to := &Model{data: s.value, driver: s.driver}
	foreign_keys = append(foreign_keys, from.typeName()+"Id", to.typeName()+"Id")

	for _, fk := range foreign_keys {
		err, from_from, foreign_value = s.getForeignKey(from, to, snakeToUpperCamel(fk))
		if err == nil {
			foreign_key = fk
			break
		}
	}

	if from_from {
		s.where(foreign_value).query()
	} else {
		query := fmt.Sprintf("%v = %v", toSnake(foreign_key), s.addToVars(foreign_value))
		s.where(query).query()
	}
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
	} else {
		s.limitStr = "1"
	}

	s.prepareQuerySql()
	if !s.hasError() {
		rows, err := s.db.Query(s.sql, s.sqlVars...)
		slog(s.sql, s.sqlVars...)
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
					if field.CanAddr() {
						values = append(values, field.Addr().Interface())
					} else {
						s.err(errors.New(fmt.Sprintf("Can't take address of %v, should be ptr", dest)))
						return
					}
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
		slog(s.sql, s.sqlVars...)
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
		slog(s.sql, s.sqlVars...)
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
				} else if dest_type.String() == "int64" {
					dest, _ = strconv.Atoi(string(dest.([]uint8)))
					dest = int64(dest.(int))
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
	case sql.NullInt64:
		return s.primaryCondiation(s.addToVars(query.(sql.NullInt64).Int64))
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
		for _, field := range m.columnsHasValue("other") {
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
			if valuer, ok := interface{}(arg).(driver.Valuer); ok {
				arg, _ = valuer.Value()
			}
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
		for _, field := range m.columnsHasValue("other") {
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
			if scanner, ok := interface{}(arg).(driver.Valuer); ok {
				arg, _ = scanner.Value()
			}
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
	for _, field := range s.model.fields("other") {
		if len(field.SqlType) > 0 {
			sqls = append(sqls, field.DbName+" "+field.SqlType)
		}
	}

	s.sql = fmt.Sprintf(
		"CREATE TABLE %v (%v)",
		s.tableName(),
		strings.Join(sqls, ","),
	)

	s.exec()
	return s
}

func (s *Do) dropTable() *Do {
	s.sql = fmt.Sprintf(
		"DROP TABLE %v",
		s.tableName(),
	)
	s.exec()
	return s
}

func (s *Do) autoMigrate() *Do {
	var table_name string
	sql := fmt.Sprintf("SELECT table_name FROM INFORMATION_SCHEMA.tables where table_name = %v", s.addToVars(s.tableName()))
	s.db.QueryRow(sql, s.sqlVars...).Scan(&table_name)
	s.sqlVars = []interface{}{}

	// If table doesn't exist
	if len(table_name) == 0 {
		s.createTable()
	} else {
		for _, field := range s.model.fields("other") {
			var column_name, data_type string
			sql := fmt.Sprintf("SELECT column_name, data_type FROM information_schema.columns WHERE table_name = %v", s.addToVars(s.tableName()))
			s.db.QueryRow(fmt.Sprintf(sql+" and column_name = %v", s.addToVars(field.DbName)), s.sqlVars...).Scan(&column_name, &data_type)
			s.sqlVars = []interface{}{}

			// If column doesn't exist
			if len(column_name) == 0 && len(field.SqlType) > 0 {
				s.sql = fmt.Sprintf("ALTER TABLE %v ADD %v %v;", s.tableName(), field.DbName, field.SqlType)
				s.exec()
			}
		}
	}
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
					for _, field := range m.columnsHasValue("other") {
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
			for _, field := range m.columnsHasValue("other") {
				m.setValueByColumn(field.DbName, field.Value, s.value)
			}
		}
	}
}
