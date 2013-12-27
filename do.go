package gorm

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"

	"github.com/jinzhu/gorm/dialect"

	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Do struct {
	db                 *DB
	search             *search
	model              *Model
	tableName          string
	value              interface{}
	usingUpdate        bool
	hasUpdate          bool
	update_attrs       map[string]interface{}
	sql                string
	sqlVars            []interface{}
	startedTransaction bool
}

func (s *Do) setSql(sql string) {
	s.sql = strings.Replace(sql, "$$", "?", -1)
}

func (s *Do) table() string {
	if len(s.tableName) == 0 {
		if len(s.search.tableName) == 0 {
			s.tableName = s.model.tableName()
		} else {
			s.tableName = s.search.tableName
		}
	}
	return s.tableName
}

func (s *Do) dialect() dialect.Dialect {
	return s.db.parent.dialect
}

func (s *Do) quote(str string) string {
	return s.dialect().Quote(str)
}

func (s *Do) err(err error) error {
	if err != nil {
		s.db.err(err)
	}
	return err
}

func (s *Do) setModel(value interface{}) *Do {
	s.model = &Model{data: value, do: s}
	s.value = value
	s.search = s.db.search
	return s
}

func (s *Do) addToVars(value interface{}) string {
	s.sqlVars = append(s.sqlVars, value)
	return s.dialect().BinVar(len(s.sqlVars))
}

func (s *Do) trace(t time.Time) {
	if len(s.sql) > 0 {
		s.db.slog(s.sql, t, s.sqlVars...)
	}
}

func (s *Do) raw(query string, values ...interface{}) *Do {
	s.setSql(s.buildWhereCondition(map[string]interface{}{"query": query, "args": values}))
	return s
}

func (s *Do) exec() *Do {
	defer s.trace(time.Now())
	if !s.db.hasError() {
		_, err := s.db.db.Exec(s.sql, s.sqlVars...)
		s.err(err)
	}
	return s
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
		columns = append(columns, s.quote(key))
		sqls = append(sqls, s.addToVars(value))
	}

	s.setSql(fmt.Sprintf(
		"INSERT INTO %v (%v) VALUES (%v) %v",
		s.table(),
		strings.Join(columns, ","),
		strings.Join(sqls, ","),
		s.dialect().ReturningStr(s.model.primaryKeyDb()),
	))
	return
}

func (s *Do) saveBeforeAssociations() {
	for _, field := range s.model.beforeAssociations() {
		do := &Do{db: s.db}

		if field.reflectValue.CanAddr() {
			do.setModel(field.reflectValue.Addr().Interface()).save()
		} else {
			// If can't take address, then clone the value and set it back
			dest_value := reflect.New(field.reflectValue.Type()).Elem()
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
				do := &Do{db: s.db}

				value := reflect_value.Index(i).Addr().Interface()
				if len(field.foreignKey) > 0 {
					s.model.setValueByColumn(field.foreignKey, s.model.primaryKeyValue(), value)
				}
				do.setModel(value).save()
			}
		default:
			do := &Do{db: s.db}
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
	defer s.trace(time.Now())
	s.model.callMethod("BeforeSave")
	s.model.callMethod("BeforeCreate")

	s.saveBeforeAssociations()
	s.prepareCreateSql()

	if !s.db.hasError() {
		var id interface{}
		if s.dialect().SupportLastInsertId() {
			if sql_result, err := s.db.db.Exec(s.sql, s.sqlVars...); s.err(err) == nil {
				id, err = sql_result.LastInsertId()
				s.err(err)
			}
		} else {
			s.err(s.db.db.QueryRow(s.sql, s.sqlVars...).Scan(&id))
		}

		if !s.db.hasError() {
			s.model.setValueByColumn(s.model.primaryKey(), id, s.value)

			s.saveAfterAssociations()
			s.model.callMethod("AfterCreate")
			s.model.callMethod("AfterSave")
		}
		return id
	}

	return
}

func (s *Do) convertToMapInterface(values interface{}) map[string]interface{} {
	attrs := map[string]interface{}{}

	switch value := values.(type) {
	case map[string]interface{}:
		for k, v := range value {
			attrs[toSnake(k)] = v
		}
	case []interface{}:
		for _, v := range value {
			for key, value := range s.convertToMapInterface(v) {
				attrs[key] = value
			}
		}
	case interface{}:
		reflect_value := reflect.ValueOf(values)

		switch reflect_value.Kind() {
		case reflect.Map:
			for _, key := range reflect_value.MapKeys() {
				attrs[toSnake(key.Interface().(string))] = reflect_value.MapIndex(key).Interface()
			}
		default:
			m := &Model{data: values, do: s}
			for _, field := range m.columnsHasValue("other") {
				attrs[field.dbName] = field.Value
			}
		}
	}
	return attrs
}

func (s *Do) updateAttrs(values interface{}, ignore_protected_attrs ...bool) *Do {
	ignore_protected := len(ignore_protected_attrs) > 0 && ignore_protected_attrs[0]
	s.usingUpdate = true

	if maps := s.convertToMapInterface(values); len(maps) > 0 {
		results, has_update := s.model.updatedColumnsAndValues(maps, ignore_protected)
		if len(results) > 0 {
			s.update_attrs = results
		}
		s.hasUpdate = has_update
	}
	return s
}

func (s *Do) prepareUpdateSql(include_self bool) {
	var sqls []string
	for key, value := range s.update_attrs {
		sqls = append(sqls, fmt.Sprintf("%v = %v", s.quote(key), s.addToVars(value)))
	}

	if include_self {
		data := s.model.reflectData()
		if data.CanAddr() {
			for key, value := range s.model.columnsAndValues("update") {
				sqls = append(sqls, fmt.Sprintf("%v = %v", s.quote(key), s.addToVars(value)))
			}
		}
	}

	s.setSql(fmt.Sprintf(
		"UPDATE %v SET %v %v",
		s.table(),
		strings.Join(sqls, ", "),
		s.combinedSql(),
	))
	return
}

func (s *Do) updateColumns(value interface{}) *Do {
	s.update_attrs = s.convertToMapInterface(value)
	s.prepareUpdateSql(false)
	if !s.db.hasError() {
		s.exec()
		s.updateAttrs(s.update_attrs)
	}
	return s
}

func (s *Do) update() *Do {
	if s.usingUpdate && !s.hasUpdate {
		return s
	}

	s.model.callMethod("BeforeSave")
	s.model.callMethod("BeforeUpdate")
	s.saveBeforeAssociations()

	s.prepareUpdateSql(true)

	if !s.db.hasError() {
		s.exec()
		s.saveAfterAssociations()

		s.model.callMethod("AfterUpdate")
		s.model.callMethod("AfterSave")
	}

	return s
}

func (s *Do) delete() *Do {
	s.model.callMethod("BeforeDelete")

	if !s.db.hasError() {
		if !s.search.unscope && s.model.hasColumn("DeletedAt") {
			s.setSql(fmt.Sprintf("UPDATE %v SET deleted_at=%v %v", s.table(), s.addToVars(time.Now()), s.combinedSql()))
		} else {
			s.setSql(fmt.Sprintf("DELETE FROM %v %v", s.table(), s.combinedSql()))
		}
		s.exec()
		s.model.callMethod("AfterDelete")
	}
	return s
}

func (s *Do) prepareQuerySql() {
	s.setSql(fmt.Sprintf("SELECT %v FROM %v %v", s.selectSql(), s.table(), s.combinedSql()))
	return
}

func (s *Do) first() *Do {
	s.search = s.search.clone().order(s.model.primaryKeyDb()).limit(1)
	s.query()
	return s
}

func (s *Do) last() *Do {
	s.search = s.search.clone().order(s.model.primaryKeyDb() + " DESC").limit(1)
	s.query()
	return s
}

func (s *Do) getForeignKey(from *Model, to *Model, foreign_key string) (err error, from_from bool, foreign_value interface{}) {
	if has_column, is_slice, value := from.columnAndValue(foreign_key); has_column {
		from_from = true
		if is_slice {
			foreign_value = to.primaryKeyValue()
		} else {
			foreign_value = value
		}
	} else if has_column, _, _ := to.columnAndValue(foreign_key); has_column {
		foreign_value = from.primaryKeyValue()
	} else {
		err = errors.New("Can't find valid foreign Key")
	}
	return
}

func (s *Do) related(value interface{}, foreign_keys ...string) *Do {
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
		query := fmt.Sprintf("%v = %v", s.quote(toSnake(foreign_key)), s.addToVars(foreign_value))
		s.where(query).query()
	}
	return s
}

func (s *Do) row() *sql.Row {
	defer s.trace(time.Now())
	s.prepareQuerySql()
	return s.db.db.QueryRow(s.sql, s.sqlVars...)
}

func (s *Do) rows() (*sql.Rows, error) {
	defer s.trace(time.Now())
	s.prepareQuerySql()
	return s.db.db.Query(s.sql, s.sqlVars...)
}

func (s *Do) query() *Do {
	defer s.trace(time.Now())
	var (
		is_slice  bool
		dest_type reflect.Type
	)
	dest_out := reflect.Indirect(reflect.ValueOf(s.value))

	if dest_out.Kind() == reflect.Slice {
		is_slice = true
		dest_type = dest_out.Type().Elem()
	} else {
		s.search = s.search.clone().limit(1)
	}

	s.prepareQuerySql()
	if !s.db.hasError() {
		rows, err := s.db.db.Query(s.sql, s.sqlVars...)

		if s.err(err) != nil {
			return s
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
			s.err(RecordNotFound)
		}
	}
	return s
}

func (s *Do) count(value interface{}) *Do {
	s.search = s.search.clone().selects("count(*)")
	s.err(s.row().Scan(value))
	return s
}

func (s *Do) pluck(column string, value interface{}) *Do {
	dest_out := reflect.Indirect(reflect.ValueOf(value))
	s.search = s.search.clone().selects(column)
	if dest_out.Kind() != reflect.Slice {
		s.err(errors.New("Results should be a slice"))
		return s
	}

	rows, err := s.rows()
	if s.err(err) == nil {
		defer rows.Close()
		for rows.Next() {
			dest := reflect.New(dest_out.Type().Elem()).Interface()
			s.err(rows.Scan(dest))
			dest_out.Set(reflect.Append(dest_out, reflect.ValueOf(dest).Elem()))
		}
	}
	return s
}

func (s *Do) primaryCondiation(value interface{}) string {
	return fmt.Sprintf("(%v = %v)", s.quote(s.model.primaryKeyDb()), value)
}

func (s *Do) buildWhereCondition(clause map[string]interface{}) (str string) {
	switch value := clause["query"].(type) {
	case string:
		if regexp.MustCompile("^\\s*\\d+\\s*$").MatchString(value) {
			id, _ := strconv.Atoi(value)
			return s.primaryCondiation(s.addToVars(id))
		} else {
			str = value
		}
	case int, int64, int32:
		return s.primaryCondiation(s.addToVars(value))
	case sql.NullInt64:
		return s.primaryCondiation(s.addToVars(value.Int64))
	case []int64, []int, []int32, []string:
		str = fmt.Sprintf("(%v in (?))", s.quote(s.model.primaryKeyDb()))
		clause["args"] = []interface{}{value}
	case map[string]interface{}:
		var sqls []string
		for key, value := range value {
			sqls = append(sqls, fmt.Sprintf("(%v = %v)", s.quote(key), s.addToVars(value)))
		}
		return strings.Join(sqls, " AND ")
	case interface{}:
		m := &Model{data: value, do: s}
		var sqls []string
		for _, field := range m.columnsHasValue("other") {
			sqls = append(sqls, fmt.Sprintf("(%v = %v)", s.quote(field.dbName), s.addToVars(field.Value)))
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
				temp_marks = append(temp_marks, s.addToVars(values.Index(i).Interface()))
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
			return fmt.Sprintf("(%v <> %v)", s.quote(s.model.primaryKeyDb()), id)
		} else if regexp.MustCompile("(?i) (=|<>|>|<|LIKE|IS) ").MatchString(value) {
			str = fmt.Sprintf(" NOT (%v) ", value)
			not_equal_sql = fmt.Sprintf("NOT (%v)", value)
		} else {
			str = fmt.Sprintf("(%v NOT IN (?))", s.quote(value))
			not_equal_sql = fmt.Sprintf("(%v <> ?)", s.quote(value))
		}
	case int, int64, int32:
		return fmt.Sprintf("(%v <> %v)", s.quote(s.model.primaryKeyDb()), value)
	case []int64, []int, []int32, []string:
		if reflect.ValueOf(value).Len() > 0 {
			str = fmt.Sprintf("(%v not in (?))", s.quote(s.model.primaryKeyDb()))
			clause["args"] = []interface{}{value}
		} else {
			return ""
		}
	case map[string]interface{}:
		var sqls []string
		for key, value := range value {
			sqls = append(sqls, fmt.Sprintf("(%v <> %v)", s.quote(key), s.addToVars(value)))
		}
		return strings.Join(sqls, " AND ")
	case interface{}:
		m := &Model{data: value, do: s}
		var sqls []string
		for _, field := range m.columnsHasValue("other") {
			sqls = append(sqls, fmt.Sprintf("(%v <> %v)", s.quote(field.dbName), s.addToVars(field.Value)))
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
				temp_marks = append(temp_marks, s.addToVars(values.Index(i).Interface()))
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

func (s *Do) where(where ...interface{}) *Do {
	if len(where) > 0 {
		s.search = s.search.clone().where(where[0], where[1:]...)
	}
	return s
}

func (s *Do) whereSql() (sql string) {
	var primary_condiations, and_conditions, or_conditions []string

	if !s.search.unscope && s.model.hasColumn("DeletedAt") {
		primary_condiations = append(primary_condiations, "(deleted_at IS NULL OR deleted_at <= '0001-01-02')")
	}

	if !s.model.primaryKeyZero() {
		primary_condiations = append(primary_condiations, s.primaryCondiation(s.addToVars(s.model.primaryKeyValue())))
	}

	for _, clause := range s.search.whereClause {
		and_conditions = append(and_conditions, s.buildWhereCondition(clause))
	}

	for _, clause := range s.search.orClause {
		or_conditions = append(or_conditions, s.buildWhereCondition(clause))
	}

	for _, clause := range s.search.notClause {
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
	if len(s.search.selectStr) == 0 {
		return "*"
	} else {
		return s.search.selectStr
	}
}

func (s *Do) orderSql() string {
	if len(s.search.orders) == 0 {
		return ""
	} else {
		return " ORDER BY " + strings.Join(s.search.orders, ",")
	}
}

func (s *Do) limitSql() string {
	if len(s.search.limitStr) == 0 {
		return ""
	} else {
		return " LIMIT " + s.search.limitStr
	}
}

func (s *Do) offsetSql() string {
	if len(s.search.offsetStr) == 0 {
		return ""
	} else {
		return " OFFSET " + s.search.offsetStr
	}
}

func (s *Do) groupSql() string {
	if len(s.search.groupStr) == 0 {
		return ""
	} else {
		return " GROUP BY " + s.search.groupStr
	}
}

func (s *Do) havingSql() string {
	if s.search.havingClause == nil {
		return ""
	} else {
		return " HAVING " + s.buildWhereCondition(s.search.havingClause)
	}
}

func (s *Do) joinsSql() string {
	return ""
}

func (s *Do) combinedSql() string {
	return s.whereSql() + s.groupSql() + s.havingSql() + s.orderSql() + s.limitSql() + s.offsetSql()
}

func (s *Do) createTable() *Do {
	var sqls []string
	for _, field := range s.model.fields("migration") {
		if len(field.sqlTag()) > 0 {
			sqls = append(sqls, s.quote(field.dbName)+" "+field.sqlTag())
		}
	}
	s.setSql(fmt.Sprintf("CREATE TABLE %v (%v)", s.table(), strings.Join(sqls, ",")))
	s.exec()
	return s
}

func (s *Do) dropTable() *Do {
	s.setSql(fmt.Sprintf("DROP TABLE %v", s.table()))
	s.exec()
	return s
}

func (s *Do) modifyColumn(column string, typ string) {
	s.setSql(fmt.Sprintf("ALTER TABLE %v MODIFY %v %v", s.table(), s.quote(column), typ))
	s.exec()
}

func (s *Do) dropColumn(column string) {
	s.setSql(fmt.Sprintf("ALTER TABLE %v DROP COLUMN %v", s.table(), s.quote(column)))
	s.exec()
}

func (s *Do) addIndex(column string, names ...string) {
	var index_name string
	if len(names) > 0 {
		index_name = names[0]
	} else {
		index_name = fmt.Sprintf("index_%v_on_%v", s.table(), column)
	}

	s.setSql(fmt.Sprintf("CREATE INDEX %v ON %v(%v);", index_name, s.table(), s.quote(column)))
	s.exec()
}

func (s *Do) removeIndex(index_name string) {
	s.setSql(fmt.Sprintf("DROP INDEX %v ON %v", index_name, s.table()))
	s.exec()
}

func (s *Do) autoMigrate() *Do {
	var table_name string
	s.setSql(fmt.Sprintf("SELECT table_name FROM INFORMATION_SCHEMA.tables where table_name = %v", s.addToVars(s.table())))
	s.db.db.QueryRow(s.sql, s.sqlVars...).Scan(&table_name)
	s.sqlVars = []interface{}{}

	// If table doesn't exist
	if len(table_name) == 0 {
		s.createTable()
	} else {
		for _, field := range s.model.fields("migration") {
			var column_name, data_type string
			s.setSql(fmt.Sprintf("SELECT column_name, data_type FROM information_schema.columns WHERE table_name = %v and column_name = %v",
				s.addToVars(s.table()),
				s.addToVars(field.dbName),
			))
			s.db.db.QueryRow(s.sql, s.sqlVars...).Scan(&column_name, &data_type)
			s.sqlVars = []interface{}{}

			// If column doesn't exist
			if len(column_name) == 0 && len(field.sqlTag()) > 0 {
				s.setSql(fmt.Sprintf("ALTER TABLE %v ADD %v %v;", s.table(), field.dbName, field.sqlTag()))
				s.exec()
			}
		}
	}
	return s
}

func (s *Do) begin() *Do {
	if db, ok := s.db.db.(sqlDb); ok {
		if tx, err := db.Begin(); err == nil {
			s.db.db = interface{}(tx).(sqlCommon)
			s.startedTransaction = true
		}
	}
	return s
}

func (s *Do) commit_or_rollback() *Do {
	if s.startedTransaction {
		if db, ok := s.db.db.(sqlTx); ok {
			if s.db.hasError() {
				db.Rollback()
			} else {
				db.Commit()
			}
			s.db.db = s.db.parent.db
		}
	}
	return s
}

func (s *Do) initialize() *Do {
	for _, clause := range s.search.whereClause {
		s.updateAttrs(clause["query"])
	}
	for _, attrs := range s.search.initAttrs {
		s.updateAttrs(attrs)
	}
	for _, attrs := range s.search.assignAttrs {
		s.updateAttrs(attrs)
	}
	return s
}
