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
	guessedTableName   string
	specifiedTableName string
	startedTransaction bool

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
		s.guessedTableName = s.model.tableName()
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

func (s *Do) setModel(value interface{}) *Do {
	s.model = &Model{data: value, do: s}
	s.value = value
	return s
}

func (s *Do) addToVars(value interface{}) string {
	s.sqlVars = append(s.sqlVars, value)
	if s.chain.driver() == "postgres" {
		return fmt.Sprintf("$%d", len(s.sqlVars))
	} else {
		return "?"
	}
}

func (s *Do) exec(sqls ...string) (err error) {
	if !s.chain.hasError() {
		if len(sqls) > 0 {
			s.sql = sqls[0]
		}

		now := time.Now()
		_, err = s.db.Exec(s.sql, s.sqlVars...)
		s.chain.slog(s.sql, now, s.sqlVars...)
	}
	return s.err(err)
}

func (s *Do) save() *Do {
	if s.model.primaryKeyZero() {
		s.create()
	} else {
		s.update()
	}
	return s
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
		do := &Do{chain: s.chain, db: s.db}

		reflect_value := reflect.ValueOf(field.Value)
		if reflect_value.CanAddr() {
			do.setModel(reflect_value.Addr().Interface()).save()
		} else {
			// If can't take address, then clone the value and set it back
			dest_value := reflect.New(reflect_value.Type()).Elem()
			m := &Model{data: field.Value, do: s}
			for _, f := range m.columnsHasValue("other") {
				dest_value.FieldByName(f.Name).Set(reflect.ValueOf(f.Value))
			}
			do.setModel(dest_value.Addr().Interface()).save()
			m.setValueByColumn(field.Name, dest_value.Interface(), s.value)
		}

		if len(field.foreignKey) > 0 {
			s.model.setValueByColumn(field.foreignKey, do.model.primaryKeyValue(), s.model.data)
		}
	}
}

func (s *Do) saveAfterAssociations() {
	for _, field := range s.model.afterAssociations() {
		reflect_value := reflect.ValueOf(field.Value)

		switch reflect_value.Kind() {
		case reflect.Slice:
			for i := 0; i < reflect_value.Len(); i++ {
				do := &Do{chain: s.chain, db: s.db}

				value := reflect_value.Index(i).Addr().Interface()
				if len(field.foreignKey) > 0 {
					s.model.setValueByColumn(field.foreignKey, s.model.primaryKeyValue(), value)
				}
				do.setModel(value).save()
			}
		default:
			do := &Do{chain: s.chain, db: s.db}
			if reflect_value.CanAddr() {
				s.model.setValueByColumn(field.foreignKey, s.model.primaryKeyValue(), field.Value)
				do.setModel(field.Value).save()
			} else {
				dest_value := reflect.New(reflect.TypeOf(field.Value)).Elem()
				m := &Model{data: field.Value, do: s}
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
	s.model.callMethod("BeforeCreate")
	s.model.callMethod("BeforeSave")

	s.saveBeforeAssociations()
	s.prepareCreateSql()

	if !s.chain.hasError() {
		var id interface{}

		now := time.Now()
		if s.chain.driver() == "postgres" {
			s.err(s.db.QueryRow(s.sql, s.sqlVars...).Scan(&id))
		} else {
			if sql_result, err := s.db.Exec(s.sql, s.sqlVars...); s.err(err) == nil {
				id, err = sql_result.LastInsertId()
				s.err(err)
			}
		}
		s.chain.slog(s.sql, now, s.sqlVars...)

		if !s.chain.hasError() {
			s.model.setValueByColumn(s.model.primaryKey(), id, s.value)

			s.saveAfterAssociations()
			s.model.callMethod("AfterCreate")
			s.model.callMethod("AfterSave")
		}
		return id
	}

	return
}

func (s *Do) setUpdateAttrs(values interface{}, ignore_protected_attrs ...bool) *Do {
	switch vs := values.(type) {
	case map[string]interface{}:
		s.updateAttrs = vs
	case []interface{}:
		for _, value := range vs {
			s.setUpdateAttrs(value, ignore_protected_attrs...)
		}
	case interface{}:
		m := &Model{data: values, do: s}
		s.updateAttrs = map[string]interface{}{}
		for _, field := range m.columnsHasValue("other") {
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

	for key, value := range s.model.columnsAndValues("update") {
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

func (s *Do) update() *Do {
	update_attrs := s.updateAttrs
	if len(update_attrs) > 0 {
		var need_update bool
		if update_attrs, need_update = s.prepareUpdateAttrs(); !need_update {
			return s
		}
	}

	s.model.callMethod("BeforeUpdate")
	s.model.callMethod("BeforeSave")

	s.saveBeforeAssociations()
	s.prepareUpdateSql(update_attrs)

	if !s.chain.hasError() {
		s.exec()
		s.saveAfterAssociations()

		s.model.callMethod("AfterUpdate")
		s.model.callMethod("AfterSave")
	}

	return s
}

func (s *Do) delete() *Do {
	s.model.callMethod("BeforeDelete")

	if !s.chain.hasError() {
		if !s.unscoped && s.model.hasColumn("DeletedAt") {
			s.sql = fmt.Sprintf("UPDATE %v SET deleted_at=%v %v", s.tableName(), s.addToVars(time.Now()), s.combinedSql())
		} else {
			s.sql = fmt.Sprintf("DELETE FROM %v %v", s.tableName(), s.combinedSql())
		}
		s.exec()
		s.model.callMethod("AfterDelete")
	}
	return s
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

	from := &Model{data: value, do: s}
	to := &Model{data: s.value, do: s}
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
	if !s.chain.hasError() {
		now := time.Now()
		rows, err := s.db.Query(s.sql, s.sqlVars...)
		s.chain.slog(s.sql, now, s.sqlVars...)

		if s.err(err) != nil {
			return
		}

		defer rows.Close()
		var has_record bool
		for rows.Next() {
			has_record = true
			dest := dest_out
			if is_slice {
				dest = reflect.New(dest_type).Elem()
			}

			columns, _ := rows.Columns()
			var values []interface{}
			for _, value := range columns {
				field := dest.FieldByName(snakeToUpperCamel(value))
				if field.IsValid() {
					values = append(values, field.Addr().Interface())
				} else {
					var ignore interface{}
					values = append(values, &ignore)
				}
			}
			s.err(rows.Scan(values...))

			if is_slice {
				dest_out.Set(reflect.Append(dest_out, dest))
			}
		}

		if !has_record && !is_slice {
			s.err(errors.New("Record not found!"))
		}
	}
}

func (s *Do) count(value interface{}) {
	s.prepareQuerySql()
	if !s.chain.hasError() {
		now := time.Now()
		s.err(s.db.QueryRow(s.sql, s.sqlVars...).Scan(value))
		s.chain.slog(s.sql, now, s.sqlVars...)
	}
}

func (s *Do) pluck(column string, value interface{}) {
	s.selectStr = column
	dest_out := reflect.Indirect(reflect.ValueOf(value))

	if dest_out.Kind() != reflect.Slice {
		s.err(errors.New("Results should be a slice"))
		return
	}

	s.prepareQuerySql()

	if !s.chain.hasError() {
		now := time.Now()
		rows, err := s.db.Query(s.sql, s.sqlVars...)
		s.chain.slog(s.sql, now, s.sqlVars...)

		if s.err(err) == nil {
			defer rows.Close()
			for rows.Next() {
				dest := reflect.New(dest_out.Type().Elem()).Interface()
				s.err(rows.Scan(dest))
				dest_out.Set(reflect.Append(dest_out, reflect.ValueOf(dest).Elem()))
			}
		}
	}
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
	switch value := clause["query"].(type) {
	case string:
		if regexp.MustCompile("^\\s*\\d+\\s*$").MatchString(value) {
			id, _ := strconv.Atoi(value)
			return s.primaryCondiation(s.addToVars(id))
		} else {
			str = "(" + value + ")"
		}
	case int, int64, int32:
		return s.primaryCondiation(s.addToVars(value))
	case sql.NullInt64:
		return s.primaryCondiation(s.addToVars(value.Int64))
	case []int64, []int, []int32, []string:
		str = fmt.Sprintf("(%v in (?))", s.model.primaryKeyDb())
		clause["args"] = []interface{}{value}
	case map[string]interface{}:
		var sqls []string
		for key, value := range value {
			sqls = append(sqls, fmt.Sprintf("(%v = %v)", key, s.addToVars(value)))
		}
		return strings.Join(sqls, " AND ")
	case interface{}:
		m := &Model{data: value, do: s}
		var sqls []string
		for _, field := range m.columnsHasValue("other") {
			sqls = append(sqls, fmt.Sprintf("(%v = %v)", field.DbName, s.addToVars(field.Value)))
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
	var not_equal_sql string

	switch value := clause["query"].(type) {
	case string:
		if regexp.MustCompile("^\\s*\\d+\\s*$").MatchString(value) {
			id, _ := strconv.Atoi(value)
			return fmt.Sprintf("(%v <> %v)", s.model.primaryKeyDb(), id)
		} else if regexp.MustCompile("(?i) (=|<>|>|<|LIKE|IS) ").MatchString(value) {
			str = fmt.Sprintf(" NOT (%v) ", value)
			not_equal_sql = fmt.Sprintf("NOT (%v)", value)
		} else {
			str = fmt.Sprintf("(%v NOT IN (?))", value)
			not_equal_sql = fmt.Sprintf("(%v <> ?)", value)
		}
	case int, int64, int32:
		return fmt.Sprintf("(%v <> %v)", s.model.primaryKeyDb(), value)
	case []int64, []int, []int32, []string:
		if reflect.ValueOf(value).Len() > 0 {
			str = fmt.Sprintf("(%v not in (?))", s.model.primaryKeyDb())
			clause["args"] = []interface{}{value}
		} else {
			return ""
		}
	case map[string]interface{}:
		var sqls []string
		for key, value := range value {
			sqls = append(sqls, fmt.Sprintf("(%v <> %v)", key, s.addToVars(value)))
		}
		return strings.Join(sqls, " AND ")
	case interface{}:
		m := &Model{data: value, do: s}
		var sqls []string
		for _, field := range m.columnsHasValue("other") {
			sqls = append(sqls, fmt.Sprintf("(%v <> %v)", field.DbName, s.addToVars(field.Value)))
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
		primary_condiations = append(primary_condiations, "(deleted_at IS NULL OR deleted_at <= '0001-01-02')")
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

	or_sql := strings.Join(or_conditions, " OR ")
	combined_sql := strings.Join(and_conditions, " AND ")
	if len(combined_sql) > 0 {
		if len(or_sql) > 0 {
			combined_sql = combined_sql + " OR " + or_sql
		}
	} else {
		combined_sql = or_sql
	}

	if len(primary_condiations) > 0 {
		sql = "WHERE " + strings.Join(primary_condiations, " AND ")
		if len(combined_sql) > 0 {
			sql = sql + " AND (" + combined_sql + ")"
		}
	} else if len(combined_sql) > 0 {
		sql = "WHERE " + combined_sql
	}
	return
}

func (s *Do) selectSql() string {
	if len(s.selectStr) == 0 {
		return "*"
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
	for _, field := range s.model.fields("migration") {
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
		for _, field := range s.model.fields("migration") {
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

func (s *Do) begin() *Do {
	if db, ok := s.db.(sql_db); ok {
		if tx, err := db.Begin(); err == nil {
			s.db = interface{}(tx).(sql_common)
			s.startedTransaction = true
		}
	}
	return s
}

func (s *Do) commit_or_rollback() {
	if s.startedTransaction {
		if db, ok := s.db.(sql_tx); ok {
			if s.chain.hasError() {
				db.Rollback()
			} else {
				db.Commit()
			}
		}
	}
}

func (s *Do) initializeWithSearchCondition() {
	for _, clause := range s.whereClause {
		switch value := clause["query"].(type) {
		case map[string]interface{}:
			for k, v := range value {
				s.model.setValueByColumn(k, v, s.value)
			}
		case []interface{}:
			for _, obj := range value {
				switch reflect.ValueOf(obj).Kind() {
				case reflect.Struct:
					m := &Model{data: obj, do: s}
					for _, field := range m.columnsHasValue("other") {
						s.model.setValueByColumn(field.DbName, field.Value, s.value)
					}
				case reflect.Map:
					for key, value := range obj.(map[string]interface{}) {
						s.model.setValueByColumn(key, value, s.value)
					}
				}
			}
		case interface{}:
			m := &Model{data: value, do: s}
			for _, field := range m.columnsHasValue("other") {
				s.model.setValueByColumn(field.DbName, field.Value, s.value)
			}
		}
	}
}
