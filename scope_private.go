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
				sqls = append(sqls, fmt.Sprintf("(%v = %v)", scope.Quote(field.DBName), scope.AddToVars(field.Field.Interface())))
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
				sqls = append(sqls, fmt.Sprintf("(%v <> %v)", scope.Quote(field.DBName), scope.AddToVars(field.Field.Interface())))
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
	if !s.Dialect().HasTop() {
		if len(s.Search.Limit) == 0 {
			return ""
		} else {
			return " LIMIT " + s.Search.Limit
		}
	} else {
		return ""
	}
}

func (s *Scope) topSql() string {
	if s.Dialect().HasTop() && len(s.Search.Offset) == 0 {
		if len(s.Search.Limit) == 0 {
			return ""
		} else {
			return " TOP(" + s.Search.Limit + ")"
		}
	} else {
		return ""
	}
}

func (s *Scope) offsetSql() string {
	if len(s.Search.Offset) == 0 {
		return ""
	} else {
		if s.Dialect().HasTop() {
			sql := " OFFSET " + s.Search.Offset + " ROW "
			if len(s.Search.Limit) > 0 {
				sql += "FETCH NEXT " + s.Search.Limit + " ROWS ONLY"
			}
			return sql
		} else {
			return " OFFSET " + s.Search.Offset
		}
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
		scope.Raw(fmt.Sprintf("SELECT %v %v FROM %v %v", scope.topSql(), scope.selectSql(), scope.QuotedTableName(), scope.CombinedConditionSql()))
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
	data := scope.IndirectValue()
	if !data.CanAddr() {
		return values, true
	}

	for key, value := range values {
		if field, ok := scope.FieldByName(SnakeToUpperCamel(key)); ok && field.Field.IsValid() {
			func() {
				defer func() {
					if err := recover(); err != nil {
						hasUpdate = true
						field.Set(value)
					}
				}()

				if field.Field.Interface() != value {
					switch field.Field.Kind() {
					case reflect.Int, reflect.Int32, reflect.Int64:
						if s, ok := value.(string); ok {
							i, err := strconv.Atoi(s)
							if scope.Err(err) == nil {
								value = i
							}
						}

						if field.Field.Int() != reflect.ValueOf(value).Int() {
							hasUpdate = true
							field.Set(value)
						}
					default:
						hasUpdate = true
						field.Set(value)
					}
				}
			}()
		}
	}
	return
}

func (scope *Scope) sqlTagForField(field *Field) (typ string) {
	if scope.db == nil {
		return ""
	}
	var size = 255

	fieldTag := field.Tag.Get(scope.db.parent.tagIdentifier)
	var setting = parseTagSetting(fieldTag)

	if value, ok := setting["SIZE"]; ok {
		if i, err := strconv.Atoi(value); err == nil {
			size = i
		} else {
			size = 0
		}
	}

	if value, ok := setting["TYPE"]; ok {
		typ = value
	}

	additionalType := setting["NOT NULL"] + " " + setting["UNIQUE"]
	if value, ok := setting["DEFAULT"]; ok {
		additionalType = additionalType + "DEFAULT " + value
	}

	value := field.Field.Interface()
	reflectValue := field.Field

	switch reflectValue.Kind() {
	case reflect.Slice:
		if _, ok := value.([]byte); !ok {
			return
		}
	case reflect.Struct:
		if field.IsScanner() {
			var getScannerValue func(reflect.Value)
			getScannerValue = func(value reflect.Value) {
				reflectValue = value
				if _, isScanner := reflect.New(reflectValue.Type()).Interface().(sql.Scanner); isScanner {
					getScannerValue(reflectValue.Field(0))
				}
			}
			getScannerValue(reflectValue.Field(0))
		} else if !field.IsTime() {
			return
		}
	}

	if len(typ) == 0 {
		if field.IsPrimaryKey {
			typ = scope.Dialect().PrimaryKeyTag(reflectValue, size)
		} else {
			typ = scope.Dialect().SqlTag(reflectValue, size)
		}
	}

	if len(additionalType) > 0 {
		typ = typ + " " + additionalType
	}
	return
}

func (scope *Scope) row() *sql.Row {
	defer scope.Trace(NowFunc())
	scope.prepareQuerySql()
	return scope.DB().QueryRow(scope.Sql, scope.SqlVars...)
}

func (scope *Scope) rows() (*sql.Rows, error) {
	defer scope.Trace(NowFunc())
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
		scope.Err(errors.New("results should be a slice"))
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
	value := scope.IndirectValue()
	if value.Kind() == reflect.Slice {
		return value.Type().Elem().Name()
	} else {
		return value.Type().Name()
	}
}

func (scope *Scope) related(value interface{}, foreignKeys ...string) *Scope {
	toScope := scope.db.NewScope(value)
	fromScopeType := scope.typeName()
	toScopeType := toScope.typeName()
	scopeType := ""

	for _, foreignKey := range append(foreignKeys, toScope.typeName()+"Id", scope.typeName()+"Id") {
		if keys := strings.Split(foreignKey, "."); len(keys) > 1 {
			scopeType = keys[0]
			foreignKey = keys[1]
		}

		if scopeType == "" || scopeType == fromScopeType {
			if field, ok := scope.FieldByName(foreignKey); ok {
				relationship := field.Relationship
				if relationship != nil && relationship.ForeignKey != "" {
					foreignKey = relationship.ForeignKey

					if relationship.Kind == "many_to_many" {
						joinSql := fmt.Sprintf(
							"INNER JOIN %v ON %v.%v = %v.%v",
							scope.Quote(relationship.JoinTable),
							scope.Quote(relationship.JoinTable),
							scope.Quote(ToSnake(relationship.AssociationForeignKey)),
							toScope.QuotedTableName(),
							scope.Quote(toScope.PrimaryKey()))
						whereSql := fmt.Sprintf("%v.%v = ?", scope.Quote(relationship.JoinTable), scope.Quote(ToSnake(relationship.ForeignKey)))
						toScope.db.Joins(joinSql).Where(whereSql, scope.PrimaryKeyValue()).Find(value)
						return scope
					}
				}

				// has one
				if foreignValue, ok := scope.FieldValueByName(foreignKey); ok {
					toScope.inlineCondition(foreignValue).callCallbacks(scope.db.parent.callback.queries)
					return scope
				}
			}
		}

		if scopeType == "" || scopeType == toScopeType {
			// has many
			if toScope.HasColumn(foreignKey) {
				sql := fmt.Sprintf("%v = ?", scope.Quote(ToSnake(foreignKey)))
				return toScope.inlineCondition(sql, scope.PrimaryKeyValue()).callCallbacks(scope.db.parent.callback.queries)
			}
		}
	}
	scope.Err(fmt.Errorf("invalid association %v", foreignKeys))
	return scope
}

func (scope *Scope) createJoinTable(field *Field) {
	if field.Relationship != nil && field.Relationship.JoinTable != "" {
		if !scope.Dialect().HasTable(scope, field.Relationship.JoinTable) {
			newScope := scope.db.NewScope("")
			primaryKeySqlType := scope.Dialect().SqlTag(reflect.ValueOf(scope.PrimaryKeyValue()), 255)
			newScope.Raw(fmt.Sprintf("CREATE TABLE %v (%v)",
				field.Relationship.JoinTable,
				strings.Join([]string{
					scope.Quote(ToSnake(field.Relationship.ForeignKey)) + " " + primaryKeySqlType,
					scope.Quote(ToSnake(field.Relationship.AssociationForeignKey)) + " " + primaryKeySqlType}, ",")),
			).Exec()
			scope.Err(newScope.db.Error)
		}
	}
}

func (scope *Scope) createTable() *Scope {
	var sqls []string
	for _, field := range scope.Fields() {
		if field.IsNormal {
			sqlTag := scope.sqlTagForField(field)
			sqls = append(sqls, scope.Quote(field.DBName)+" "+sqlTag)
		}
		scope.createJoinTable(field)
	}
	scope.Raw(fmt.Sprintf("CREATE TABLE %v (%v)", scope.QuotedTableName(), strings.Join(sqls, ","))).Exec()
	return scope
}

func (scope *Scope) dropTable() *Scope {
	scope.Raw(fmt.Sprintf("DROP TABLE %v", scope.QuotedTableName())).Exec()
	return scope
}

func (scope *Scope) dropTableIfExists() *Scope {
	scope.Raw(fmt.Sprintf("DROP TABLE IF EXISTS %v", scope.QuotedTableName())).Exec()
	return scope
}

func (scope *Scope) modifyColumn(column string, typ string) {
	scope.Raw(fmt.Sprintf("ALTER TABLE %v MODIFY %v %v", scope.QuotedTableName(), scope.Quote(column), typ)).Exec()
}

func (scope *Scope) dropColumn(column string) {
	scope.Raw(fmt.Sprintf("ALTER TABLE %v DROP COLUMN %v", scope.QuotedTableName(), scope.Quote(column))).Exec()
}

func (scope *Scope) addIndex(unique bool, indexName string, column ...string) {
	var columns []string
	for _, name := range column {
		columns = append(columns, scope.Quote(name))
	}

	sqlCreate := "CREATE INDEX"
	if unique {
		sqlCreate = "CREATE UNIQUE INDEX"
	}

	scope.Raw(fmt.Sprintf("%s %v ON %v(%v);", sqlCreate, indexName, scope.QuotedTableName(), strings.Join(columns, ", "))).Exec()
}

func (scope *Scope) removeIndex(indexName string) {
	scope.Dialect().RemoveIndex(scope, indexName)
}

func (scope *Scope) autoMigrate() *Scope {
	tableName := scope.TableName()
	quotedTableName := scope.QuotedTableName()

	if !scope.Dialect().HasTable(scope, tableName) {
		scope.createTable()
	} else {
		for _, field := range scope.Fields() {
			if !scope.Dialect().HasColumn(scope, tableName, field.DBName) {
				if field.IsNormal {
					sqlTag := scope.sqlTagForField(field)
					scope.Raw(fmt.Sprintf("ALTER TABLE %v ADD %v %v;", quotedTableName, field.DBName, sqlTag)).Exec()
				}
			}
			scope.createJoinTable(field)
		}
	}
	return scope
}
