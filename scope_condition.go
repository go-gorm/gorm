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

func (scope *Scope) Quote(str string) string {
	return scope.Dialect().Quote(str)
}

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
			var temp_marks []string
			for i := 0; i < values.Len(); i++ {
				temp_marks = append(temp_marks, scope.AddToVars(values.Index(i).Interface()))
			}
			str = strings.Replace(str, "?", strings.Join(temp_marks, ","), 1)
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
	var not_equal_sql string

	switch value := clause["query"].(type) {
	case string:
		if regexp.MustCompile("^\\s*\\d+\\s*$").MatchString(value) {
			id, _ := strconv.Atoi(value)
			return fmt.Sprintf("(%v <> %v)", scope.Quote(scope.PrimaryKey()), id)
		} else if regexp.MustCompile("(?i) (=|<>|>|<|LIKE|IS) ").MatchString(value) {
			str = fmt.Sprintf(" NOT (%v) ", value)
			not_equal_sql = fmt.Sprintf("NOT (%v)", value)
		} else {
			str = fmt.Sprintf("(%v NOT IN (?))", scope.Quote(value))
			not_equal_sql = fmt.Sprintf("(%v <> ?)", scope.Quote(value))
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
			var temp_marks []string
			for i := 0; i < values.Len(); i++ {
				temp_marks = append(temp_marks, scope.AddToVars(values.Index(i).Interface()))
			}
			str = strings.Replace(str, "?", strings.Join(temp_marks, ","), 1)
		default:
			if scanner, ok := interface{}(arg).(driver.Valuer); ok {
				arg, _ = scanner.Value()
			}
			str = strings.Replace(not_equal_sql, "?", scope.AddToVars(arg), 1)
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
	var primary_condiations, and_conditions, or_conditions []string

	if !scope.Search.unscope && scope.HasColumn("DeletedAt") {
		primary_condiations = append(primary_condiations, "(deleted_at IS NULL OR deleted_at <= '0001-01-02')")
	}

	if !scope.PrimaryKeyZero() {
		primary_condiations = append(primary_condiations, scope.primaryCondiation(scope.AddToVars(scope.PrimaryKeyValue())))
	}

	for _, clause := range scope.Search.whereClause {
		and_conditions = append(and_conditions, scope.buildWhereCondition(clause))
	}

	for _, clause := range scope.Search.orClause {
		or_conditions = append(or_conditions, scope.buildWhereCondition(clause))
	}

	for _, clause := range scope.Search.notClause {
		and_conditions = append(and_conditions, scope.buildNotCondition(clause))
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

func (s *Scope) selectSql() string {
	if len(s.Search.selectStr) == 0 {
		return "*"
	} else {
		return s.Search.selectStr
	}
}

func (s *Scope) orderSql() string {
	if len(s.Search.orders) == 0 {
		return ""
	} else {
		return " ORDER BY " + strings.Join(s.Search.orders, ",")
	}
}

func (s *Scope) limitSql() string {
	if len(s.Search.limitStr) == 0 {
		return ""
	} else {
		return " LIMIT " + s.Search.limitStr
	}
}

func (s *Scope) offsetSql() string {
	if len(s.Search.offsetStr) == 0 {
		return ""
	} else {
		return " OFFSET " + s.Search.offsetStr
	}
}

func (s *Scope) groupSql() string {
	if len(s.Search.groupStr) == 0 {
		return ""
	} else {
		return " GROUP BY " + s.Search.groupStr
	}
}

func (s *Scope) havingSql() string {
	if s.Search.havingClause == nil {
		return ""
	} else {
		return " HAVING " + s.buildWhereCondition(s.Search.havingClause)
	}
}

func (s *Scope) joinsSql() string {
	return s.Search.joinsStr + " "
}

func (scope *Scope) prepareQuerySql() {
	if scope.Search.raw {
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
