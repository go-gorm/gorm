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
