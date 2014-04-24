package gorm

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"go/ast"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func (scope *Scope) primaryCondiation(value interface{}) string {
	return fmt.Sprintf("(%v = %v)", scope.Quote(scope.PrimaryKey()), value)
}

func (scope *Scope) buildWhereCondition(clause map[string]interface{}) (str string) {
	switch value := clause["query"].(type) {
	case string:
		// if string is number
		if regexp.MustCompile("^\\s*\\d+\\s*$").MatchString(value) {
			id, _ := strconv.Atoi(value)
			return scope.primaryCondiation(scope.AddToVars(id))
		} else {
			str = value
		}
	case int, int64, int32:
		return scope.primaryCondiation(scope.AddToVars(value))
	case sql.NullInt64:
		return scope.primaryCondiation(scope.AddToVars(value.Int64))
	case []int64, []int, []int32, []string:
		str = fmt.Sprintf("(%v in (?))", scope.Quote(scope.PrimaryKey()))
		clause["args"] = []interface{}{value}
	case map[string]interface{}:
		var sqls []string
		for key, value := range value {
			sqls = append(sqls, fmt.Sprintf("(%v = %v)", scope.Quote(key), scope.AddToVars(value)))
		}
		return strings.Join(sqls, " AND ")
	case interface{}:
		var sqls []string
		for _, field := range scope.New(value).Fields() {
			if !field.IsBlank {
				sqls = append(sqls, fmt.Sprintf("(%v = %v)", scope.Quote(field.DBName), scope.AddToVars(field.Value)))
			}
		}
		return strings.Join(sqls, " AND ")
	}

	args := clause["args"].([]interface{})
	for _, arg := range args {
		switch reflect.TypeOf(arg).Kind() {
		case reflect.Slice: // For where("id in (?)", []int64{1,2})
			values := reflect.ValueOf(arg)
			var tempMarks []string
			for i := 0; i < values.Len(); i++ {
				tempMarks = append(tempMarks, scope.AddToVars(values.Index(i).Interface()))
			}
			str = strings.Replace(str, "?", strings.Join(tempMarks, ","), 1)
		default:
			if valuer, ok := interface{}(arg).(driver.Valuer); ok {
				arg, _ = valuer.Value()
			}

			str = strings.Replace(str, "?", scope.AddToVars(arg), 1)
		}
	}
	return
}

func (scope *Scope) buildNotCondition(clause map[string]interface{}) (str string) {
	var notEqualSql string

	switch value := clause["query"].(type) {
	case string:
		if regexp.MustCompile("^\\s*\\d+\\s*$").MatchString(value) {
			id, _ := strconv.Atoi(value)
			return fmt.Sprintf("(%v <> %v)", scope.Quote(scope.PrimaryKey()), id)
		} else if regexp.MustCompile("(?i) (=|<>|>|<|LIKE|IS) ").MatchString(value) {
			str = fmt.Sprintf(" NOT (%v) ", value)
			notEqualSql = fmt.Sprintf("NOT (%v)", value)
		} else {
			str = fmt.Sprintf("(%v NOT IN (?))", scope.Quote(value))
			notEqualSql = fmt.Sprintf("(%v <> ?)", scope.Quote(value))
		}
	case int, int64, int32:
		return fmt.Sprintf("(%v <> %v)", scope.Quote(scope.PrimaryKey()), value)
	case []int64, []int, []int32, []string:
		if reflect.ValueOf(value).Len() > 0 {
			str = fmt.Sprintf("(%v not in (?))", scope.Quote(scope.PrimaryKey()))
			clause["args"] = []interface{}{value}
		} else {
			return ""
		}
	case map[string]interface{}:
		var sqls []string
		for key, value := range value {
			sqls = append(sqls, fmt.Sprintf("(%v <> %v)", scope.Quote(key), scope.AddToVars(value)))
		}
		return strings.Join(sqls, " AND ")
	case interface{}:
		var sqls []string
		for _, field := range scope.New(value).Fields() {
			if !field.IsBlank {
				sqls = append(sqls, fmt.Sprintf("(%v <> %v)", scope.Quote(field.DBName), scope.AddToVars(field.Value)))
			}
		}
		return strings.Join(sqls, " AND ")
	}

	args := clause["args"].([]interface{})
	for _, arg := range args {
		switch reflect.TypeOf(arg).Kind() {
		case reflect.Slice: // For where("id in (?)", []int64{1,2})
			values := reflect.ValueOf(arg)
			var tempMarks []string
			for i := 0; i < values.Len(); i++ {
				tempMarks = append(tempMarks, scope.AddToVars(values.Index(i).Interface()))
			}
			str = strings.Replace(str, "?", strings.Join(tempMarks, ","), 1)
		default:
			if scanner, ok := interface{}(arg).(driver.Valuer); ok {
				arg, _ = scanner.Value()
			}
			str = strings.Replace(notEqualSql, "?", scope.AddToVars(arg), 1)
		}
	}
	return
}

func (scope *Scope) where(where ...interface{}) {
	if len(where) > 0 {
		scope.Search = scope.Search.clone().where(where[0], where[1:]...)
	}
}

func (scope *Scope) whereSql() (sql string) {
	var primaryCondiations, andConditions, orConditions []string

	if !scope.Search.Unscope && scope.HasColumn("DeletedAt") {
		primaryCondiations = append(primaryCondiations, "(deleted_at IS NULL OR deleted_at <= '0001-01-02')")
	}

	if !scope.PrimaryKeyZero() {
		primaryCondiations = append(primaryCondiations, scope.primaryCondiation(scope.AddToVars(scope.PrimaryKeyValue())))
	}

	for _, clause := range scope.Search.WhereConditions {
		andConditions = append(andConditions, scope.buildWhereCondition(clause))
	}

	for _, clause := range scope.Search.OrConditions {
		orConditions = append(orConditions, scope.buildWhereCondition(clause))
	}

	for _, clause := range scope.Search.NotConditions {
		andConditions = append(andConditions, scope.buildNotCondition(clause))
	}

	orSql := strings.Join(orConditions, " OR ")
	combinedSql := strings.Join(andConditions, " AND ")
	if len(combinedSql) > 0 {
		if len(orSql) > 0 {
			combinedSql = combinedSql + " OR " + orSql
		}
	} else {
		combinedSql = orSql
	}

	if len(primaryCondiations) > 0 {
		sql = "WHERE " + strings.Join(primaryCondiations, " AND ")
		if len(combinedSql) > 0 {
			sql = sql + " AND (" + combinedSql + ")"
		}
	} else if len(combinedSql) > 0 {
		sql = "WHERE " + combinedSql
	}
	return
}

func (s *Scope) selectSql() string {
	if len(s.Search.Select) == 0 {
		return "*"
	} else {
		return s.Search.Select
	}
}

func (s *Scope) orderSql() string {
	if len(s.Search.Orders) == 0 {
		return ""
	} else {
		return " ORDER BY " + strings.Join(s.Search.Orders, ",")
	}
}

func (s *Scope) limitSql() string {
	if len(s.Search.Limit) == 0 {
		return ""
	} else {
		return " LIMIT " + s.Search.Limit
	}
}

func (s *Scope) offsetSql() string {
	if len(s.Search.Offset) == 0 {
		return ""
	} else {
		return " OFFSET " + s.Search.Offset
	}
}

func (s *Scope) groupSql() string {
	if len(s.Search.Group) == 0 {
		return ""
	} else {
		return " GROUP BY " + s.Search.Group
	}
}

func (s *Scope) havingSql() string {
	if s.Search.HavingCondition == nil {
		return ""
	} else {
		return " HAVING " + s.buildWhereCondition(s.Search.HavingCondition)
	}
}

func (s *Scope) joinsSql() string {
	return s.Search.Joins + " "
}

func (scope *Scope) prepareQuerySql() {
	if scope.Search.Raw {
		scope.Raw(strings.TrimLeft(scope.CombinedConditionSql(), "WHERE "))
	} else {
		scope.Raw(fmt.Sprintf("SELECT %v FROM %v %v", scope.selectSql(), scope.TableName(), scope.CombinedConditionSql()))
	}
	return
}

func (scope *Scope) inlineCondition(values ...interface{}) *Scope {
	if len(values) > 0 {
		scope.Search = scope.Search.clone().where(values[0], values[1:]...)
	}
	return scope
}

func (scope *Scope) callCallbacks(funcs []*func(s *Scope)) *Scope {
	for _, f := range funcs {
		(*f)(scope)
		if scope.skipLeft {
			break
		}
	}
	return scope
}

func (scope *Scope) updatedAttrsWithValues(values map[string]interface{}, ignoreProtectedAttrs bool) (results map[string]interface{}, hasUpdate bool) {
	data := reflect.Indirect(reflect.ValueOf(scope.Value))
	if !data.CanAddr() {
		return values, true
	}

	for key, value := range values {
		if field := data.FieldByName(snakeToUpperCamel(key)); field.IsValid() {
			func() {
				defer func() {
					if err := recover(); err != nil {
						hasUpdate = true
						setFieldValue(field, value)
					}
				}()

				if field.Interface() != value {
					switch field.Kind() {
					case reflect.Int, reflect.Int32, reflect.Int64:
						if s, ok := value.(string); ok {
							i, err := strconv.Atoi(s)
							if scope.Err(err) == nil {
								value = i
							}
						}

						if field.Int() != reflect.ValueOf(value).Int() {
							hasUpdate = true
							setFieldValue(field, value)
						}
					default:
						hasUpdate = true
						setFieldValue(field, value)
					}
				}
			}()
		}
	}
	return
}

func (scope *Scope) sqlTagForField(field *Field) (tag string) {
	tag, addationalTag, size := parseSqlTag(field.Tag.Get(scope.db.parent.tagIdentifier))

	if tag == "-" {
		field.IsIgnored = true
	}

	value := field.Value
	reflectValue := reflect.ValueOf(value)

	switch reflectValue.Kind() {
	case reflect.Slice:
		if _, ok := value.([]byte); !ok {
			return
		}
	case reflect.Struct:
		if field.IsScanner() {
			reflectValue = reflectValue.Field(0)
		} else if !field.IsTime() {
			return
		}
	}

	if len(tag) == 0 {
		if field.isPrimaryKey {
			tag = scope.Dialect().PrimaryKeyTag(reflectValue, size)
		} else {
			tag = scope.Dialect().SqlTag(reflectValue, size)
		}
	}

	if len(addationalTag) > 0 {
		tag = tag + " " + addationalTag
	}
	return
}

func (scope *Scope) row() *sql.Row {
	defer scope.Trace(time.Now())
	scope.prepareQuerySql()
	return scope.DB().QueryRow(scope.Sql, scope.SqlVars...)
}

func (scope *Scope) rows() (*sql.Rows, error) {
	defer scope.Trace(time.Now())
	scope.prepareQuerySql()
	return scope.DB().Query(scope.Sql, scope.SqlVars...)
}

func (scope *Scope) initialize() *Scope {
	for _, clause := range scope.Search.WhereConditions {
		scope.updatedAttrsWithValues(convertInterfaceToMap(clause["query"]), false)
	}
	scope.updatedAttrsWithValues(convertInterfaceToMap(scope.Search.InitAttrs), false)
	scope.updatedAttrsWithValues(convertInterfaceToMap(scope.Search.AssignAttrs), false)
	return scope
}

func (scope *Scope) pluck(column string, value interface{}) *Scope {
	dest := reflect.Indirect(reflect.ValueOf(value))
	scope.Search = scope.Search.clone().selects(column)
	if dest.Kind() != reflect.Slice {
		scope.Err(errors.New("Results should be a slice"))
		return scope
	}

	rows, err := scope.rows()
	if scope.Err(err) == nil {
		defer rows.Close()
		for rows.Next() {
			elem := reflect.New(dest.Type().Elem()).Interface()
			scope.Err(rows.Scan(elem))
			dest.Set(reflect.Append(dest, reflect.ValueOf(elem).Elem()))
		}
	}
	return scope
}

func (scope *Scope) count(value interface{}) *Scope {
	scope.Search = scope.Search.clone().selects("count(*)")
	scope.Err(scope.row().Scan(value))
	return scope
}

func (scope *Scope) typeName() string {
	value := reflect.Indirect(reflect.ValueOf(scope.Value))
	if value.Kind() == reflect.Slice {
		return value.Type().Elem().Name()
	} else {
		return value.Type().Name()
	}
}

func (scope *Scope) related(value interface{}, foreignKeys ...string) *Scope {
	toScope := scope.db.NewScope(value)

	for _, foreignKey := range append(foreignKeys, toScope.typeName()+"Id", scope.typeName()+"Id") {
		if foreignValue, ok := scope.FieldByName(foreignKey); ok {
			return toScope.inlineCondition(foreignValue).callCallbacks(scope.db.parent.callback.queries)
		} else if toScope.HasColumn(foreignKey) {
			sql := fmt.Sprintf("%v = ?", scope.Quote(toSnake(foreignKey)))
			return toScope.inlineCondition(sql, scope.PrimaryKeyValue()).callCallbacks(scope.db.parent.callback.queries)
		}
	}
	return scope
}

func (scope *Scope) createTable() *Scope {
	var sqls []string
	for _, field := range scope.Fields() {
		if !field.IsIgnored && len(field.SqlTag) > 0 {
			sqls = append(sqls, scope.Quote(field.DBName)+" "+field.SqlTag)
		}
	}
	scope.Raw(fmt.Sprintf("CREATE TABLE %v (%v)", scope.TableName(), strings.Join(sqls, ","))).Exec()
	return scope
}

func (scope *Scope) dropTable() *Scope {
	scope.Raw(fmt.Sprintf("DROP TABLE %v", scope.TableName())).Exec()
	return scope
}

func (scope *Scope) modifyColumn(column string, typ string) {
	scope.Raw(fmt.Sprintf("ALTER TABLE %v MODIFY %v %v", scope.TableName(), scope.Quote(column), typ)).Exec()
}

func (scope *Scope) dropColumn(column string) {
	scope.Raw(fmt.Sprintf("ALTER TABLE %v DROP COLUMN %v", scope.TableName(), scope.Quote(column))).Exec()
}

func (scope *Scope) addIndex(column string, names ...string) {
	var indexName string
	if len(names) > 0 {
		indexName = names[0]
	} else {
		indexName = fmt.Sprintf("index_%v_on_%v", scope.TableName(), column)
	}

	scope.Raw(fmt.Sprintf("CREATE INDEX %v ON %v(%v);", indexName, scope.TableName(), scope.Quote(column))).Exec()
}

func (scope *Scope) removeIndex(indexName string) {
	scope.Raw(fmt.Sprintf("DROP INDEX %v ON %v", indexName, scope.TableName())).Exec()
}

func (scope *Scope) autoMigrate() *Scope {
	// scope.db.source sample: root:@/testdatabase?parseTime=true
	from := strings.Index(scope.db.source, "/")
	to := strings.Index(scope.db.source, "?")
	if to == -1 {
		to = len(scope.db.source)
	}
	databaseName := scope.db.source[from:to]

	var tableName string
	scope.Raw(fmt.Sprintf("SELECT table_name FROM INFORMATION_SCHEMA.tables where table_schema = %v AND table_name = %v",
		scope.AddToVars(databaseName),
		scope.AddToVars(scope.TableName())))
	scope.DB().QueryRow(scope.Sql, scope.SqlVars...).Scan(&tableName)
	scope.SqlVars = []interface{}{}

	// If table doesn't exist
	if len(tableName) == 0 {
		scope.createTable()
	} else {
		for _, field := range scope.Fields() {
			var column, data string
			scope.Raw(fmt.Sprintf("SELECT column_name, data_type FROM information_schema.columns WHERE table_schema = %v AND table_name = %v AND column_name = %v",
				scope.AddToVars(databaseName),
				scope.AddToVars(scope.TableName()),
				scope.AddToVars(field.DBName),
			))
			scope.DB().QueryRow(scope.Sql, scope.SqlVars...).Scan(&column, &data)
			scope.SqlVars = []interface{}{}

			// If column doesn't exist
			if len(column) == 0 && len(field.SqlTag) > 0 && !field.IsIgnored {
				scope.Raw(fmt.Sprintf("ALTER TABLE %v ADD %v %v;", scope.TableName(), field.DBName, field.SqlTag)).Exec()
			}
		}
	}
	return scope
}

func (scope *Scope) getPrimaryKey() string {
	var indirectValue reflect.Value

	indirectValue = reflect.Indirect(reflect.ValueOf(scope.Value))

	if indirectValue.Kind() == reflect.Slice {
		indirectValue = reflect.New(indirectValue.Type().Elem()).Elem()
	}

	if !indirectValue.IsValid() {
		return "id"
	}

	scopeTyp := indirectValue.Type()
	for i := 0; i < scopeTyp.NumField(); i++ {
		fieldStruct := scopeTyp.Field(i)
		if !ast.IsExported(fieldStruct.Name) {
			continue
		}

		// if primaryKey tag found, return column name
		if fieldStruct.Tag.Get("primaryKey") != "" {
			return toSnake(fieldStruct.Name)
		}
	}

	//If primaryKey tag not found, fallback to id
	return "id"
}
