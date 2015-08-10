package gorm

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

func (scope *Scope) primaryCondition(value interface{}) string {
	return fmt.Sprintf("(%v = %v)", scope.Quote(scope.PrimaryKey()), value)
}

func (scope *Scope) buildWhereCondition(clause map[string]interface{}) (str string) {
	switch value := clause["query"].(type) {
	case string:
		// if string is number
		if regexp.MustCompile("^\\s*\\d+\\s*$").MatchString(value) {
			id, _ := strconv.Atoi(value)
			return scope.primaryCondition(scope.AddToVars(id))
		} else if value != "" {
			str = fmt.Sprintf("(%v)", value)
		}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, sql.NullInt64:
		return scope.primaryCondition(scope.AddToVars(value))
	case []int, []int8, []int16, []int32, []int64, []uint, []uint8, []uint16, []uint32, []uint64, []string, []interface{}:
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
			if !field.IsIgnored && !field.IsBlank {
				sqls = append(sqls, fmt.Sprintf("(%v = %v)", scope.Quote(field.DBName), scope.AddToVars(field.Field.Interface())))
			}
		}
		return strings.Join(sqls, " AND ")
	}

	args := clause["args"].([]interface{})
	for _, arg := range args {
		switch reflect.ValueOf(arg).Kind() {
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
	var primaryKey = scope.PrimaryKey()

	switch value := clause["query"].(type) {
	case string:
		// is number
		if regexp.MustCompile("^\\s*\\d+\\s*$").MatchString(value) {
			id, _ := strconv.Atoi(value)
			return fmt.Sprintf("(%v <> %v)", scope.Quote(primaryKey), id)
		} else if regexp.MustCompile("(?i) (=|<>|>|<|LIKE|IS) ").MatchString(value) {
			str = fmt.Sprintf(" NOT (%v) ", value)
			notEqualSql = fmt.Sprintf("NOT (%v)", value)
		} else {
			str = fmt.Sprintf("(%v NOT IN (?))", scope.Quote(value))
			notEqualSql = fmt.Sprintf("(%v <> ?)", scope.Quote(value))
		}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, sql.NullInt64:
		return fmt.Sprintf("(%v <> %v)", scope.Quote(primaryKey), value)
	case []int, []int8, []int16, []int32, []int64, []uint, []uint8, []uint16, []uint32, []uint64, []string:
		if reflect.ValueOf(value).Len() > 0 {
			str = fmt.Sprintf("(%v NOT IN (?))", scope.Quote(primaryKey))
			clause["args"] = []interface{}{value}
		}
		return ""
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
		switch reflect.ValueOf(arg).Kind() {
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

func (scope *Scope) buildSelectQuery(clause map[string]interface{}) (str string) {
	switch value := clause["query"].(type) {
	case string:
		str = value
	case []string:
		str = strings.Join(value, ", ")
	}

	args := clause["args"].([]interface{})
	for _, arg := range args {
		switch reflect.ValueOf(arg).Kind() {
		case reflect.Slice:
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

func (scope *Scope) whereSql() (sql string) {
	var primaryConditions, andConditions, orConditions []string

	if !scope.Search.Unscoped && scope.Fields()["deleted_at"] != nil {
		sql := fmt.Sprintf("(%v.deleted_at IS NULL OR %v.deleted_at <= '0001-01-02')", scope.QuotedTableName(), scope.QuotedTableName())
		primaryConditions = append(primaryConditions, sql)
	}

	if !scope.PrimaryKeyZero() {
		primaryConditions = append(primaryConditions, scope.primaryCondition(scope.AddToVars(scope.PrimaryKeyValue())))
	}

	for _, clause := range scope.Search.whereConditions {
		if sql := scope.buildWhereCondition(clause); sql != "" {
			andConditions = append(andConditions, sql)
		}
	}

	for _, clause := range scope.Search.orConditions {
		if sql := scope.buildWhereCondition(clause); sql != "" {
			orConditions = append(orConditions, sql)
		}
	}

	for _, clause := range scope.Search.notConditions {
		if sql := scope.buildNotCondition(clause); sql != "" {
			andConditions = append(andConditions, sql)
		}
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

	if len(primaryConditions) > 0 {
		sql = "WHERE " + strings.Join(primaryConditions, " AND ")
		if len(combinedSql) > 0 {
			sql = sql + " AND (" + combinedSql + ")"
		}
	} else if len(combinedSql) > 0 {
		sql = "WHERE " + combinedSql
	}
	return
}

var hasCountRegexp = regexp.MustCompile(`(?i)count(.+)`)

func (scope *Scope) selectSql() string {
	if len(scope.Search.selects) == 0 {
		return "*"
	}
	sql := scope.buildSelectQuery(scope.Search.selects)
	scope.Search.countingQuery = hasCountRegexp.MatchString(sql)
	return scope.buildSelectQuery(scope.Search.selects)
}

func (scope *Scope) orderSql() string {
	if len(scope.Search.orders) == 0 || scope.Search.countingQuery {
		return ""
	}
	return " ORDER BY " + strings.Join(scope.Search.orders, ",")
}

func (scope *Scope) limitSql() string {
	if !scope.Dialect().HasTop() {
		if len(scope.Search.limit) == 0 {
			return ""
		}
		return " LIMIT " + scope.Search.limit
	}

	return ""
}

func (scope *Scope) topSql() string {
	if scope.Dialect().HasTop() && len(scope.Search.offset) == 0 {
		if len(scope.Search.limit) == 0 {
			return ""
		}
		return " TOP(" + scope.Search.limit + ")"
	}

	return ""
}

func (scope *Scope) offsetSql() string {
	if len(scope.Search.offset) == 0 {
		return ""
	}

	if scope.Dialect().HasTop() {
		sql := " OFFSET " + scope.Search.offset + " ROW "
		if len(scope.Search.limit) > 0 {
			sql += "FETCH NEXT " + scope.Search.limit + " ROWS ONLY"
		}
		return sql
	}
	return " OFFSET " + scope.Search.offset
}

func (scope *Scope) groupSql() string {
	if len(scope.Search.group) == 0 {
		return ""
	}
	return " GROUP BY " + scope.Search.group
}

func (scope *Scope) havingSql() string {
	if scope.Search.havingConditions == nil {
		return ""
	}

	var andConditions []string

	for _, clause := range scope.Search.havingConditions {
		if sql := scope.buildWhereCondition(clause); sql != "" {
			andConditions = append(andConditions, sql)
		}
	}

	combinedSql := strings.Join(andConditions, " AND ")
	if len(combinedSql) == 0 {
		return ""
	}

	return " HAVING " + combinedSql
}

func (scope *Scope) joinsSql() string {
	return scope.Search.joins + " "
}

func (scope *Scope) prepareQuerySql() {
	if scope.Search.raw {
		scope.Raw(strings.TrimSuffix(strings.TrimPrefix(scope.CombinedConditionSql(), " WHERE ("), ")"))
	} else {
		scope.Raw(fmt.Sprintf("SELECT %v %v FROM %v %v", scope.topSql(), scope.selectSql(), scope.QuotedTableName(), scope.CombinedConditionSql()))
	}
	return
}

func (scope *Scope) inlineCondition(values ...interface{}) *Scope {
	if len(values) > 0 {
		scope.Search.Where(values[0], values[1:]...)
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
	if !scope.IndirectValue().CanAddr() {
		return values, true
	}

	var hasExpr bool
	fields := scope.Fields()
	for key, value := range values {
		if field, ok := fields[ToDBName(key)]; ok && field.Field.IsValid() {
			if !reflect.DeepEqual(field.Field, reflect.ValueOf(value)) {
				if _, ok := value.(*expr); ok {
					hasExpr = true
				} else if !equalAsString(field.Field.Interface(), value) {
					hasUpdate = true
					field.Set(value)
				}
			}
		}
	}
	if hasExpr {
		var updateMap = map[string]interface{}{}
		for key, value := range fields {
			if v, ok := values[key]; ok {
				updateMap[key] = v
			} else {
				updateMap[key] = value.Field.Interface()
			}
		}
		return updateMap, true
	}
	return
}

func (scope *Scope) row() *sql.Row {
	defer scope.Trace(NowFunc())
	scope.callCallbacks(scope.db.parent.callback.rowQueries)
	scope.prepareQuerySql()
	return scope.SqlDB().QueryRow(scope.Sql, scope.SqlVars...)
}

func (scope *Scope) rows() (*sql.Rows, error) {
	defer scope.Trace(NowFunc())
	scope.callCallbacks(scope.db.parent.callback.rowQueries)
	scope.prepareQuerySql()
	return scope.SqlDB().Query(scope.Sql, scope.SqlVars...)
}

func (scope *Scope) initialize() *Scope {
	for _, clause := range scope.Search.whereConditions {
		scope.updatedAttrsWithValues(convertInterfaceToMap(clause["query"]), false)
	}
	scope.updatedAttrsWithValues(convertInterfaceToMap(scope.Search.initAttrs), false)
	scope.updatedAttrsWithValues(convertInterfaceToMap(scope.Search.assignAttrs), false)
	return scope
}

func (scope *Scope) pluck(column string, value interface{}) *Scope {
	dest := reflect.Indirect(reflect.ValueOf(value))
	scope.Search.Select(column)
	if dest.Kind() != reflect.Slice {
		scope.Err(fmt.Errorf("results should be a slice, not %s", dest.Kind()))
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
	scope.Search.Select("count(*)")
	scope.Err(scope.row().Scan(value))
	return scope
}

func (scope *Scope) typeName() string {
	value := scope.IndirectValue()
	if value.Kind() == reflect.Slice {
		return value.Type().Elem().Name()
	}

	return value.Type().Name()
}

func (scope *Scope) related(value interface{}, foreignKeys ...string) *Scope {
	toScope := scope.db.NewScope(value)
	fromFields := scope.Fields()
	toFields := toScope.Fields()
	for _, foreignKey := range append(foreignKeys, toScope.typeName()+"Id", scope.typeName()+"Id") {
		var fromField, toField *Field
		if field, ok := scope.FieldByName(foreignKey); ok {
			fromField = field
		} else {
			fromField = fromFields[ToDBName(foreignKey)]
		}
		if field, ok := toScope.FieldByName(foreignKey); ok {
			toField = field
		} else {
			toField = toFields[ToDBName(foreignKey)]
		}

		if fromField != nil {
			if relationship := fromField.Relationship; relationship != nil {
				if relationship.Kind == "many_to_many" {
					joinTableHandler := relationship.JoinTableHandler
					scope.Err(joinTableHandler.JoinWith(joinTableHandler, toScope.db, scope.Value).Find(value).Error)
				} else if relationship.Kind == "belongs_to" {
					query := toScope.db
					for idx, foreignKey := range relationship.ForeignDBNames {
						if field, ok := scope.FieldByName(foreignKey); ok {
							query = query.Where(fmt.Sprintf("%v = ?", scope.Quote(relationship.AssociationForeignDBNames[idx])), field.Field.Interface())
						}
					}
					scope.Err(query.Find(value).Error)
				} else if relationship.Kind == "has_many" || relationship.Kind == "has_one" {
					query := toScope.db
					for idx, foreignKey := range relationship.ForeignDBNames {
						if field, ok := scope.FieldByName(relationship.AssociationForeignDBNames[idx]); ok {
							query = query.Where(fmt.Sprintf("%v = ?", scope.Quote(foreignKey)), field.Field.Interface())
						}
					}

					if relationship.PolymorphicType != "" {
						query = query.Where(fmt.Sprintf("%v = ?", scope.Quote(relationship.PolymorphicDBName)), scope.TableName())
					}
					scope.Err(query.Find(value).Error)
				}
			} else {
				sql := fmt.Sprintf("%v = ?", scope.Quote(toScope.PrimaryKey()))
				scope.Err(toScope.db.Where(sql, fromField.Field.Interface()).Find(value).Error)
			}
			return scope
		} else if toField != nil {
			sql := fmt.Sprintf("%v = ?", scope.Quote(toField.DBName))
			scope.Err(toScope.db.Where(sql, scope.PrimaryKeyValue()).Find(value).Error)
			return scope
		}
	}

	scope.Err(fmt.Errorf("invalid association %v", foreignKeys))
	return scope
}

/**
  Return the table options string or an empty string if the table options does not exist
*/
func (scope *Scope) getTableOptions() string {
	tableOptions, ok := scope.Get("gorm:table_options")
	if !ok {
		return ""
	}
	return tableOptions.(string)
}

func (scope *Scope) createJoinTable(field *StructField) {
	if relationship := field.Relationship; relationship != nil && relationship.JoinTableHandler != nil {
		joinTableHandler := relationship.JoinTableHandler
		joinTable := joinTableHandler.Table(scope.db)
		if !scope.Dialect().HasTable(scope, joinTable) {
			toScope := &Scope{Value: reflect.New(field.Struct.Type).Interface()}

			var sqlTypes []string
			for idx, fieldName := range relationship.ForeignFieldNames {
				if field, ok := scope.Fields()[fieldName]; ok {
					value := reflect.Indirect(reflect.New(field.Struct.Type))
					primaryKeySqlType := scope.Dialect().SqlTag(value, 255, false)
					sqlTypes = append(sqlTypes, scope.Quote(relationship.ForeignDBNames[idx])+" "+primaryKeySqlType)
				}
			}

			for idx, fieldName := range relationship.AssociationForeignFieldNames {
				if field, ok := toScope.Fields()[fieldName]; ok {
					value := reflect.Indirect(reflect.New(field.Struct.Type))
					primaryKeySqlType := scope.Dialect().SqlTag(value, 255, false)
					sqlTypes = append(sqlTypes, scope.Quote(relationship.AssociationForeignDBNames[idx])+" "+primaryKeySqlType)
				}
			}
			scope.Err(scope.NewDB().Exec(fmt.Sprintf("CREATE TABLE %v (%v) %s", scope.Quote(joinTable), strings.Join(sqlTypes, ","), scope.getTableOptions())).Error)
		}
		scope.NewDB().Table(joinTable).AutoMigrate(joinTableHandler)
	}
}

func (scope *Scope) createTable() *Scope {
	var tags []string
	var primaryKeys []string
	for _, field := range scope.GetStructFields() {
		if field.IsNormal {
			sqlTag := scope.generateSqlTag(field)
			tags = append(tags, scope.Quote(field.DBName)+" "+sqlTag)
		}

		if field.IsPrimaryKey {
			primaryKeys = append(primaryKeys, scope.Quote(field.DBName))
		}
		scope.createJoinTable(field)
	}

	var primaryKeyStr string
	if len(primaryKeys) > 0 {
		primaryKeyStr = fmt.Sprintf(", PRIMARY KEY (%v)", strings.Join(primaryKeys, ","))
	}
	scope.Raw(fmt.Sprintf("CREATE TABLE %v (%v %v) %s", scope.QuotedTableName(), strings.Join(tags, ","), primaryKeyStr, scope.getTableOptions())).Exec()
	return scope
}

func (scope *Scope) dropTable() *Scope {
	scope.Raw(fmt.Sprintf("DROP TABLE %v", scope.QuotedTableName())).Exec()
	return scope
}

func (scope *Scope) dropTableIfExists() *Scope {
	if scope.Dialect().HasTable(scope, scope.TableName()) {
		scope.dropTable()
	}
	return scope
}

func (scope *Scope) modifyColumn(column string, typ string) {
	scope.Raw(fmt.Sprintf("ALTER TABLE %v MODIFY %v %v", scope.QuotedTableName(), scope.Quote(column), typ)).Exec()
}

func (scope *Scope) dropColumn(column string) {
	scope.Raw(fmt.Sprintf("ALTER TABLE %v DROP COLUMN %v", scope.QuotedTableName(), scope.Quote(column))).Exec()
}

func (scope *Scope) addIndex(unique bool, indexName string, column ...string) {
	if scope.Dialect().HasIndex(scope, scope.TableName(), indexName) {
		return
	}

	var columns []string
	for _, name := range column {
		columns = append(columns, scope.QuoteIfPossible(name))
	}

	sqlCreate := "CREATE INDEX"
	if unique {
		sqlCreate = "CREATE UNIQUE INDEX"
	}

	scope.Raw(fmt.Sprintf("%s %v ON %v(%v);", sqlCreate, indexName, scope.QuotedTableName(), strings.Join(columns, ", "))).Exec()
}

func (scope *Scope) addForeignKey(field string, dest string, onDelete string, onUpdate string) {
	var table = scope.TableName()
	var keyName = fmt.Sprintf("%s_%s_%s_foreign", table, field, regexp.MustCompile("[^a-zA-Z]").ReplaceAllString(dest, "_"))
	keyName = regexp.MustCompile("_+").ReplaceAllString(keyName, "_")
	var query = `ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s ON DELETE %s ON UPDATE %s;`
	scope.Raw(fmt.Sprintf(query, scope.QuotedTableName(), scope.QuoteIfPossible(keyName), scope.QuoteIfPossible(field), dest, onDelete, onUpdate)).Exec()
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
		for _, field := range scope.GetStructFields() {
			if !scope.Dialect().HasColumn(scope, tableName, field.DBName) {
				if field.IsNormal {
					sqlTag := scope.generateSqlTag(field)
					scope.Raw(fmt.Sprintf("ALTER TABLE %v ADD %v %v;", quotedTableName, scope.Quote(field.DBName), sqlTag)).Exec()
				}
			}
			scope.createJoinTable(field)
		}
	}

	scope.autoIndex()
	return scope
}

func (scope *Scope) autoIndex() *Scope {
	var indexes = map[string][]string{}
	var uniqueIndexes = map[string][]string{}

	for _, field := range scope.GetStructFields() {
		sqlSettings := parseTagSetting(field.Tag.Get("sql"))
		if name, ok := sqlSettings["INDEX"]; ok {
			if name == "INDEX" {
				name = fmt.Sprintf("idx_%v_%v", scope.TableName(), field.DBName)
			}
			indexes[name] = append(indexes[name], field.DBName)
		}

		if name, ok := sqlSettings["UNIQUE_INDEX"]; ok {
			if name == "UNIQUE_INDEX" {
				name = fmt.Sprintf("uix_%v_%v", scope.TableName(), field.DBName)
			}
			uniqueIndexes[name] = append(uniqueIndexes[name], field.DBName)
		}
	}

	for name, columns := range indexes {
		scope.addIndex(false, name, columns...)
	}

	for name, columns := range uniqueIndexes {
		scope.addIndex(true, name, columns...)
	}

	return scope
}
