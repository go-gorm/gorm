package gorm

import (
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

	if _, ok := value.(*expr); ok {
		e.expr = "(" + e.expr + " " + operator + " (?))"
	} else {
		e.expr = "(" + e.expr + " " + operator + " ?)"
	}
	e.args = append(e.args, value)

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

func (e *expr) Count() string {
	return "COUNT(" + e.expr + ")"
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

	return db.Update(sets)
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

