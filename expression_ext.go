package gorm

import (
	"fmt"
	"reflect"
	"strings"
)

type jexpr struct {
	expr string
	args []interface{}
}

func join(joinType string, db *DB, model interface{}, alias ...string) *jexpr {
	var al string
	if len(alias) > 0 {
		al = alias[0]
	}

	if val, ok := model.(*expr); ok {
		return &jexpr{expr: " " + joinType + " JOIN (" + val.expr + ") " + al, args: val.args}
	}
	return &jexpr{expr: " " + joinType + " JOIN " + db.T(model) + " " + al}
}

func (db *DB) InnerJoin(model interface{}, alias ...string) *jexpr {
	return join("INNER", db, model, alias...)
}

func (db *DB) LeftJoin(model interface{}, alias ...string) *jexpr {
	return join("LEFT", db, model, alias...)
}

func (db *DB) RightJoin(model interface{}, alias ...string) *jexpr {
	return join("RIGHT", db, model, alias...)
}

func (db *DB) OuterJoin(model interface{}, alias ...string) *jexpr {
	return join("OUTER", db, model, alias...)
}

func (je *jexpr) On(col1 *expr, col2 *expr) *expr {
	return &expr{expr: je.expr + " ON " + col1.expr + " = " + col2.expr, args: je.args}
}

func (je *jexpr) OnExp(e2 *expr) *expr {
	e := &expr{expr: je.expr + " ON " + e2.expr, args: je.args}
	e.args = append(e.args, e2.args...)
	return e
}

func (db *DB) L(model interface{}, name string) *expr {
	scope := db.NewScope(model)
	field, _ := scope.FieldByName(name)
	return &expr{expr: scope.Quote(scope.TableName()) + "." + scope.Quote(field.DBName)}
}

func (db *DB) LA(model interface{}, alias string, name string) *expr {
	scope := db.NewScope(model)
	field, _ := scope.FieldByName(name)
	return &expr{expr: scope.Quote(alias) + "." + scope.Quote(field.DBName)}
}

func (db *DB) QuoteExpr(table string, column string) *expr {
	scope := db.NewScope(nil)
	return &expr{expr: scope.Quote(table) + "." + scope.Quote(column)}
}

func (db *DB) C(model interface{}, names ...string) string {
	columns := make([]string, 0)

	scope := db.NewScope(model)
	for _, name := range names {
		field, _ := scope.FieldByName(name)
		columns = append(columns, field.DBName)
	}

	return strings.Join(columns, ", ")
}

func (db *DB) CA(model interface{}, alias string, names ...string) string {
	columns := make([]string, 0)

	for _, name := range names {
		columns = append(columns, db.LA(model, alias, name).expr)
	}

	return strings.Join(columns, ", ")
}

func (db *DB) CQ(model interface{}, names ...string) string {
	columns := make([]string, 0)

	for _, name := range names {
		columns = append(columns, db.L(model, name).expr)
	}

	return strings.Join(columns, ", ")
}

func (db *DB) T(model interface{}) string {
	scope := db.NewScope(model)
	return scope.TableName()
}

func (db *DB) QT(model interface{}) string {
	scope := db.NewScope(model)
	return scope.QuotedTableName()
}

func (e *expr) operator(operator string, value interface{}) *expr {
	if value == nil {
		e.expr = "(" + e.expr + " " + operator + " )"
		return e
	}

	if vExpr, ok := value.(*expr); ok {
		e.expr = "(" + e.expr + " " + operator + " (" + vExpr.expr + "))"
		e.args = append(e.args, vExpr.args...)
	} else {
		e.expr = "(" + e.expr + " " + operator + " ?)"
		e.args = append(e.args, value)
	}

	return e
}

// Union will create a statement which unions the statement of e and stmt
func (e *expr) Union(stmt *expr) *expr {
	e.expr = e.expr + " UNION " + stmt.expr
	e.args = append(e.args, stmt.args...)
	return e
}

// Union will create a statement which unions all given statements.
// stmts have to be *gorm.expr variables (but it is interface{}, so it
// can be used by external packages...)
func Union(stmts ...interface{}) *expr {
	var result *expr

	for idx, stmt := range stmts {
		if idx == 0 {
			result = stmt.(*expr)
		} else {
			result = result.Union(stmt.(*expr))
		}
	}

	return result
}

// And will create a statement which "ands" all given statements.
// stmts have to be *gorm.expr variables (but it is interface{}, so it
// can be used by external packages...)
func And(stmts ...interface{}) *expr {
	var result *expr

	for idx, stmt := range stmts {
		if idx == 0 {
			result = stmt.(*expr)
		} else {
			result = result.And(stmt.(*expr))
		}
	}

	return result
}

// Or will create a statement which "ors" all given statements.
// stmts have to be *gorm.expr variables (but it is interface{}, so it
// can be used by external packages...)
func Or(stmts ...interface{}) *expr {
	var result *expr

	for idx, stmt := range stmts {
		if idx == 0 {
			result = stmt.(*expr)
		} else {
			result = result.Or(stmt.(*expr))
		}
	}

	return result
}

// Not negates the given statement by surrounding its expression with "NOT (expr)"
// stmt has to be a *gorm.expr (but it is interface{}, so it
// can be used by external packages...)
func Not(stmt interface{}) *expr {
	e := stmt.(*expr)
	e.expr = "NOT (" + e.expr + ")"
	return e
}

// Concat will create an expression which concats all given statements together.
func Concat(stmts ...interface{}) *expr {
	e := &expr{expr: "CONCAT("}

	for i, stmt := range stmts {
		if i != 0 {
			e.expr += ", "
		}

		if exp, ok := stmt.(*expr); ok {
			e.expr += exp.expr
			e.args = append(e.args, exp.args...)
		} else {
			e.expr += "?"
			e.args = append(e.args, stmt)
		}
	}

	e.expr += ")"
	return e
}

func (e *expr) Gt(value interface{}) *expr {
	return e.operator(">", value)
}

func (e *expr) Ge(value interface{}) *expr {
	return e.operator(">=", value)
}

func (e *expr) Lt(value interface{}) *expr {
	return e.operator("<", value)
}

func (e *expr) Le(value interface{}) *expr {
	return e.operator("<=", value)
}

func (e *expr) BAnd(value interface{}) *expr {
	return e.operator("&", value)
}

func (e *expr) BOr(value interface{}) *expr {
	return e.operator("|", value)
}

func (e *expr) Like(value interface{}) *expr {
	return e.operator("LIKE", value)
}

func (e *expr) NotLike(value interface{}) *expr {
	return e.operator("NOT LIKE", value)
}

func (e *expr) Eq(value interface{}) *expr {
	if value == nil {
		return e.operator("IS NULL", value)
	} else if val := reflect.ValueOf(value); val.Kind() == reflect.Ptr && val.IsNil() {
		return e.operator("IS NULL", nil)
	}

	return e.operator("=", value)
}

func (e *expr) Neq(value interface{}) *expr {
	if value == nil {
		return e.operator("IS NOT NULL", value)
	}

	return e.operator("!=", value)
}

func (e *expr) Sum() string {
	return "SUM(" + e.expr + ")"
}

func (e *expr) SumExpr() *expr {
	e.expr = "SUM(" + e.expr + ")"
	return e
}

func (db *DB) GroupConcatExpr(e *expr, separator string, orderExpr *expr) *expr {
	e.args = append(e.args, orderExpr.args...)

	dbType := db.Dialect().GetName()
	switch dbType {
	case "mysql":
		e.expr = fmt.Sprintf("GROUP_CONCAT(%s %s SEPARATOR '%s')", e.expr, orderExpr.expr, separator)
	case "mssql":
		e.expr = fmt.Sprintf("STRING_AGG(%s, '%s') WITHIN GROUP (%s)", e.expr, separator, orderExpr.expr)
	case "sqlite3":
		e.expr = fmt.Sprintf("GROUP_CONCAT(%s,'%s')", e.expr, separator)
	case "postgres":
		e.expr = fmt.Sprintf("string_agg(%s, '%s' %s)", e.expr, separator, orderExpr.expr)
	case "oracle":
		e.expr = fmt.Sprintf("LISTAGG(%s, '%s') WITHIN GROUP (%s)", e.expr, separator, orderExpr.expr)
	default:
		panic(fmt.Sprintf("Unsuported database type %s for GroupConcat!", dbType))
	}
	return e
}

func (db *DB) GroupConcat(e *expr, separator string, orderExpr *expr) string {
	return db.GroupConcatExpr(e, separator, orderExpr).expr
}

func Order(stmts ...interface{}) *expr {
	e := &expr{expr: "ORDER BY "}
	for i, stmt := range stmts {
		if i != 0 {
			e.expr += ", "
		}
		if exp, ok := stmt.(*expr); ok {
			e.expr += exp.expr
			e.args = append(e.args, exp.args...)
		} else {
			e.expr += "?"
			e.args = append(e.args, stmt)
		}
	}
	return e
}

func (e *expr) Max() string {
	return "MAX(" + e.expr + ")"
}

func (e *expr) Min() string {
	return "MIN(" + e.expr + ")"
}

func (e *expr) Count() string {
	return "COUNT(" + e.expr + ")"
}

func (e *expr) CountExpr() *expr {
	e.expr = "COUNT(" + e.expr + ")"
	return e
}

func (e *expr) Distinct() *expr {
	e.expr = "DISTINCT " + e.expr
	return e
}

func (e *expr) DistinctColumn() string {
	return "DISTINCT " + e.expr
}

func (e *expr) in(operator string, values ...interface{}) *expr {
	// NOTE: Maybe there is a better way to do this? :)
	if len(values) == 1 {
		s := reflect.ValueOf(values[0])
		if s.Kind() == reflect.Slice {
			vals := make([]interface{}, s.Len())
			qm := make([]string, s.Len())

			for i := 0; i < s.Len(); i++ {
				vals[i] = s.Index(i).Interface()
				qm[i] = "?"
			}

			e.expr = "(" + e.expr + operator + " IN (" + strings.Join(qm, ",") + "))"
			e.args = append(e.args, vals...)
			return e
		}
		if vexpr, ok := values[0].(*expr); ok {
			e.expr = "(" + e.expr + operator + " IN (" + vexpr.expr + "))"
			e.args = append(e.args, vexpr.args...)
			return e
		}
	}

	qm := make([]string, len(values))
	for i := 0; i < len(values); i++ {
		qm[i] = "?"
	}

	e.expr = "(" + e.expr + operator + " IN (" + strings.Join(qm, ",") + "))"
	e.args = append(e.args, values...)
	return e
}

func (e *expr) In(values ...interface{}) *expr {
	return e.in("", values...)
}

func (e *expr) NotIn(values ...interface{}) *expr {
	return e.in(" NOT", values...)
}

func (e *expr) OrderAsc() string {
	return e.expr + " ASC "
}

func (e *expr) OrderDesc() string {
	return e.expr + " DESC "
}

func (e *expr) OrderAscExpr() *expr {
	e.expr = e.expr + " ASC "
	return e
}

func (e *expr) OrderDescExpr() *expr {
	e.expr = e.expr + " DESC "
	return e
}

func (e *expr) Or(e2 *expr) *expr {
	e.expr = "(" + e.expr + " OR " + e2.expr + ")"
	e.args = append(e.args, e2.args...)

	return e
}

func (e *expr) And(e2 *expr) *expr {
	e.expr = "(" + e.expr + " AND " + e2.expr + ")"
	e.args = append(e.args, e2.args...)

	return e
}

func (db *DB) UpdateFields(fields ...string) *DB {
	sets := make(map[string]interface{})
	m := reflect.ValueOf(db.Value).Elem()
	for _, field := range fields {
		sets[db.C(db.Value, field)] = m.FieldByName(field).Interface()
	}

	return db.Set("gorm:save_associations", false).Update(sets)
}

func (db *DB) SelectFields(fields ...string) *DB {
	selects := strings.Join(fields, ", ")

	return db.Select(selects)
}

func (e *expr) Intersect(e2 *expr) *expr {
	e.expr = "((" + e.expr + ") INTERSECT (" + e2.expr + "))"
	e.args = append(e.args, e2.args...)

	return e
}

func (e *expr) Alias(alias string) *expr {
	e.expr = e.expr + " " + alias + " "

	return e
}

func (db *DB) FormatDate(e *expr, format string) *expr {
	return db.Dialect().FormatDate(e, format)
}

func (db *DB) FormatDateColumn(e *expr, format string) string {
	return db.FormatDate(e, format).expr
}

func (db *DB) GetSQL() string {
	scope := db.NewScope(db.Value)

	scope.prepareQuerySQL()

	stmt := strings.ReplaceAll(scope.SQL, "$$$", "?")
	for _, arg := range scope.SQLVars {
		stmt = strings.Replace(stmt, "?", "'"+escape(fmt.Sprintf("%v", arg))+"'", 1)
	}

	return stmt
}

func (db *DB) GetSQLWhereClause() string {
	scope := db.NewScope(db.Value)

	stmt := strings.ReplaceAll(scope.CombinedConditionSql(), "$$$", "?")

	for _, arg := range scope.SQLVars {
		stmt = strings.Replace(stmt, "?", "'"+escape(fmt.Sprintf("%v", arg))+"'", 1)
	}

	return stmt
}

func escape(source string) string {
	var j int = 0
	if len(source) == 0 {
		return ""
	}
	tempStr := source[:]
	desc := make([]byte, len(tempStr)*2)
	for i := 0; i < len(tempStr); i++ {
		flag := false
		var escape byte
		switch tempStr[i] {
		case '\r':
			flag = true
			escape = '\r'

		case '\n':
			flag = true
			escape = '\n'

		case '\\':
			flag = true
			escape = '\\'

		case '\'':
			flag = true
			escape = '\''

		case '"':
			flag = true
			escape = '"'

		case '\032':
			flag = true
			escape = 'Z'

		default:
		}
		if flag {
			desc[j] = '\\'
			desc[j+1] = escape
			j = j + 2
		} else {
			desc[j] = tempStr[i]
			j = j + 1
		}
	}
	return string(desc[0:j])
}
