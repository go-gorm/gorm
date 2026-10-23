package clause

import (
	"database/sql"
	"database/sql/driver"
	"go/ast"
	"reflect"
)

// Expression renders SQL into a Builder without requiring a named clause slot.
// Expressions compose predicates, projections, values, and subqueries; Build must bind
// data separately from SQL text.
type Expression interface {
	// Build appends SQL and bound values to the builder. Implementations must keep
	// untrusted data out of directly written SQL.
	Build(builder Builder)
}

// NegationExpressionBuilder renders an expression's inverse operator when used by Not.
// Implementing it avoids a generic NOT wrapper and permits SQL-specific NULL handling.
type NegationExpressionBuilder interface {
	// NegationBuild writes the inverse predicate, including appropriate NULL/list behavior
	// for the expression type.
	NegationBuild(builder Builder)
}

// Expr is a trusted SQL template with positional ? parameters in Vars. Values are bound
// through Builder, while nested expressions render their own SQL. This is not a SQL
// parser or sanitizer; do not concatenate untrusted data into SQL.
type Expr struct {
	// SQL is the trusted template. Placeholder processing is byte-based and does not parse
	// SQL string literals or comments.
	SQL string
	// Vars supplies positional values for ? placeholders. Values may themselves be
	// expressions, so keep their SQL text trusted too.
	Vars []interface{}
	// WithoutParentheses expands slice arguments as comma-separated values without adding a
	// surrounding pair of parentheses, useful inside templates that supply their own
	// grouping.
	WithoutParentheses bool
}

// Build replaces available ? placeholders with bound values, expanding lists
// immediately after an opening parenthesis or when WithoutParentheses is set. Remaining
// variables are recorded as unnamed NamedArgs. It does not validate placeholder counts
// or parse quoted SQL literals.
func (expr Expr) Build(builder Builder) {
	var (
		afterParenthesis bool
		idx              int
	)

	for _, v := range []byte(expr.SQL) {
		if v == '?' && len(expr.Vars) > idx {
			if afterParenthesis || expr.WithoutParentheses {
				processValue(builder, expr.Vars[idx])
			} else {
				builder.AddVar(builder, expr.Vars[idx])
			}

			idx++
		} else {
			if v == '(' {
				afterParenthesis = true
			} else {
				afterParenthesis = false
			}
			builder.WriteByte(v)
		}
	}

	if idx < len(expr.Vars) {
		for _, v := range expr.Vars[idx:] {
			builder.AddVar(builder, sql.NamedArg{Value: v})
		}
	}
}

// NamedExpr renders trusted SQL containing @name parameters, resolved from sql.NamedArg
// values, string-keyed maps, or exported struct fields in Vars. It also handles
// positional ? values; unresolved names remain in the SQL text.
type NamedExpr struct {
	// SQL is the trusted template containing @name or ? references; scanning is not a full
	// SQL lexer.
	SQL string
	// Vars provides named arguments, maps, or structs. Later occurrences of a name replace
	// earlier values in the binding map.
	Vars []interface{}
}

// Build collects named values and substitutes recognized @name references using
// Builder.AddVar. Unknown names remain literal. Lists immediately after an opening
// parenthesis are expanded; SQL templates must be trusted because quoted strings are
// not parsed as protected regions.
func (expr NamedExpr) Build(builder Builder) {
	var (
		idx              int
		inName           bool
		afterParenthesis bool
		namedMap         = make(map[string]interface{}, len(expr.Vars))
	)

	for _, v := range expr.Vars {
		switch value := v.(type) {
		case sql.NamedArg:
			namedMap[value.Name] = value.Value
		case map[string]interface{}:
			for k, v := range value {
				namedMap[k] = v
			}
		default:
			var appendFieldsToMap func(reflect.Value)
			appendFieldsToMap = func(reflectValue reflect.Value) {
				reflectValue = reflect.Indirect(reflectValue)
				switch reflectValue.Kind() {
				case reflect.Struct:
					modelType := reflectValue.Type()
					for i := 0; i < modelType.NumField(); i++ {
						if fieldStruct := modelType.Field(i); ast.IsExported(fieldStruct.Name) {
							namedMap[fieldStruct.Name] = reflectValue.Field(i).Interface()

							if fieldStruct.Anonymous {
								appendFieldsToMap(reflectValue.Field(i))
							}
						}
					}
				}
			}

			appendFieldsToMap(reflect.ValueOf(value))
		}
	}

	name := make([]byte, 0, 10)

	for _, v := range []byte(expr.SQL) {
		if v == '@' && !inName {
			inName = true
			name = name[:0]
		} else if v == ' ' || v == ',' || v == ')' || v == '"' || v == '\'' || v == '`' || v == '\r' || v == '\n' || v == ';' {
			if inName {
				if nv, ok := namedMap[string(name)]; ok {
					if afterParenthesis {
						processValue(builder, nv)
					} else {
						builder.AddVar(builder, nv)
					}
				} else {
					builder.WriteByte('@')
					builder.WriteString(string(name))
				}
				inName = false
			}

			afterParenthesis = false
			builder.WriteByte(v)
		} else if v == '?' && len(expr.Vars) > idx {
			if afterParenthesis {
				processValue(builder, expr.Vars[idx])
			} else {
				builder.AddVar(builder, expr.Vars[idx])
			}

			idx++
		} else if inName {
			name = append(name, v)
		} else {
			if v == '(' {
				afterParenthesis = true
			} else {
				afterParenthesis = false
			}
			builder.WriteByte(v)
		}
	}

	if inName {
		if nv, ok := namedMap[string(name)]; ok {
			builder.AddVar(builder, nv)
		} else {
			builder.WriteByte('@')
			builder.WriteString(string(name))
		}
	}
}

// processValue binds driver.Valuer as a single value, otherwise expands slices/arrays
// into comma-separated bindings. An empty list binds nil. The caller supplies any
// surrounding parentheses.
func processValue(builder Builder, value interface{}) {
	if _, ok := value.(driver.Valuer); ok {
		builder.AddVar(builder, value)
		return
	}

	switch rv := reflect.ValueOf(value); rv.Kind() {
	case reflect.Slice, reflect.Array:
		if rv.Len() == 0 {
			builder.AddVar(builder, nil)
		} else {
			for i := 0; i < rv.Len(); i++ {
				if i > 0 {
					builder.WriteByte(',')
				}
				builder.AddVar(builder, rv.Index(i).Interface())
			}
		}
	default:
		builder.AddVar(builder, value)
	}
}

// IN tests a quoted column against Values. Empty Values render IN (NULL); a single
// scalar renders equality and multiple values render IN. Negation has distinct
// empty-list behavior, so choose application semantics before constructing an empty
// predicate.
type IN struct {
	// Column identifies the compared column or structured Column; it is rendered through
	// WriteQuoted.
	Column interface{}
	// Values is the candidate set; nested []interface{} values can represent composite
	// tuples.
	Values []interface{}
}

// Build writes IN (NULL) for no values, equality for one non-tuple value, or an IN list
// with bound values. SQL NULL semantics still apply.
func (in IN) Build(builder Builder) {
	builder.WriteQuoted(in.Column)

	switch len(in.Values) {
	case 0:
		builder.WriteString(" IN (NULL)")
	case 1:
		if _, ok := in.Values[0].([]interface{}); !ok {
			builder.WriteString(" = ")
			builder.AddVar(builder, in.Values[0])
			break
		}

		fallthrough
	default:
		builder.WriteString(" IN (")
		builder.AddVar(builder, in.Values...)
		builder.WriteByte(')')
	}
}

// NegationBuild writes IS NOT NULL for an empty set, inequality for one non-tuple
// value, or NOT IN for multiple values. It is not an unconditional true predicate for
// an empty set.
func (in IN) NegationBuild(builder Builder) {
	builder.WriteQuoted(in.Column)
	switch len(in.Values) {
	case 0:
		builder.WriteString(" IS NOT NULL")
	case 1:
		if _, ok := in.Values[0].([]interface{}); !ok {
			builder.WriteString(" <> ")
			builder.AddVar(builder, in.Values[0])
			break
		}

		fallthrough
	default:
		builder.WriteString(" NOT IN (")
		builder.AddVar(builder, in.Values...)
		builder.WriteByte(')')
	}
}

// Eq compares a quoted column with Value. Nil and nil-valued driver values use IS NULL;
// supported slice types use IN, with an empty slice rendered as IN (NULL). Other values
// use equality with a bound parameter.
type Eq struct {
	// Column identifies the left-hand identifier, rendered through WriteQuoted.
	Column interface{}
	// Value is bound as data unless it is an expression; nil and supported slices select
	// specialized predicate forms.
	Value interface{}
}

// Build renders equality, IS NULL, or IN according to Value. Supported lists are slices
// of string, int/int32/int64, uint/uint32/uint64, or interface{}; other values follow
// ordinary binding.
func (eq Eq) Build(builder Builder) {
	builder.WriteQuoted(eq.Column)

	switch eq.Value.(type) {
	case []string, []int, []int32, []int64, []uint, []uint32, []uint64, []interface{}:
		rv := reflect.ValueOf(eq.Value)
		if rv.Len() == 0 {
			builder.WriteString(" IN (NULL)")
		} else {
			builder.WriteString(" IN (")
			for i := 0; i < rv.Len(); i++ {
				if i > 0 {
					builder.WriteByte(',')
				}
				builder.AddVar(builder, rv.Index(i).Interface())
			}
			builder.WriteByte(')')
		}
	default:
		if eqNil(eq.Value) {
			builder.WriteString(" IS NULL")
		} else {
			builder.WriteString(" = ")
			builder.AddVar(builder, eq.Value)
		}
	}
}

// NegationBuild delegates to Neq, including its NULL and list behavior.
func (eq Eq) NegationBuild(builder Builder) {
	Neq(eq).Build(builder)
}

// Neq is Eq's inverse form: scalar <> comparison, IS NOT NULL for nil, or NOT IN for
// supported slices. An empty supported slice renders NOT IN (), whose validity and
// behavior depend on the database; handle empty inputs deliberately.
type Neq Eq

// Build writes inequality, IS NOT NULL, or a NOT IN list based on Value. Unlike
// IN.NegationBuild, an empty supported slice is emitted as an empty list.
func (neq Neq) Build(builder Builder) {
	builder.WriteQuoted(neq.Column)

	switch neq.Value.(type) {
	case []string, []int, []int32, []int64, []uint, []uint32, []uint64, []interface{}:
		builder.WriteString(" NOT IN (")
		rv := reflect.ValueOf(neq.Value)
		for i := 0; i < rv.Len(); i++ {
			if i > 0 {
				builder.WriteByte(',')
			}
			builder.AddVar(builder, rv.Index(i).Interface())
		}
		builder.WriteByte(')')
	default:
		if eqNil(neq.Value) {
			builder.WriteString(" IS NOT NULL")
		} else {
			builder.WriteString(" <> ")
			builder.AddVar(builder, neq.Value)
		}
	}
}

// NegationBuild delegates to Eq and its NULL/list behavior.
func (neq Neq) NegationBuild(builder Builder) {
	Eq(neq).Build(builder)
}

// Gt compares a quoted column using > and a bound Value. It has Eq's field layout but
// does not use Eq's NULL or list special cases; comparison with NULL follows SQL
// three-valued logic.
type Gt Eq

// Build writes the quoted column, >, and Value through AddVar. It does not execute a
// query.
func (gt Gt) Build(builder Builder) {
	builder.WriteQuoted(gt.Column)
	builder.WriteString(" > ")
	builder.AddVar(builder, gt.Value)
}

// NegationBuild delegates to Lte to render the inverse comparison operator.
func (gt Gt) NegationBuild(builder Builder) {
	Lte(gt).Build(builder)
}

// Gte compares a quoted column using >= and a bound Value. It has Eq's field layout but
// does not use Eq's NULL or list special cases; comparison with NULL follows SQL
// three-valued logic.
type Gte Eq

// Build writes the quoted column, >=, and Value through AddVar. It does not execute a
// query.
func (gte Gte) Build(builder Builder) {
	builder.WriteQuoted(gte.Column)
	builder.WriteString(" >= ")
	builder.AddVar(builder, gte.Value)
}

// NegationBuild delegates to Lt to render the inverse comparison operator.
func (gte Gte) NegationBuild(builder Builder) {
	Lt(gte).Build(builder)
}

// Lt compares a quoted column using < and a bound Value. It has Eq's field layout but
// does not use Eq's NULL or list special cases; comparison with NULL follows SQL
// three-valued logic.
type Lt Eq

// Build writes the quoted column, <, and Value through AddVar. It does not execute a
// query.
func (lt Lt) Build(builder Builder) {
	builder.WriteQuoted(lt.Column)
	builder.WriteString(" < ")
	builder.AddVar(builder, lt.Value)
}

// NegationBuild delegates to Gte to render the inverse comparison operator.
func (lt Lt) NegationBuild(builder Builder) {
	Gte(lt).Build(builder)
}

// Lte compares a quoted column using <= and a bound Value. It has Eq's field layout but
// does not use Eq's NULL or list special cases; comparison with NULL follows SQL
// three-valued logic.
type Lte Eq

// Build writes the quoted column, <=, and Value through AddVar. It does not execute a
// query.
func (lte Lte) Build(builder Builder) {
	builder.WriteQuoted(lte.Column)
	builder.WriteString(" <= ")
	builder.AddVar(builder, lte.Value)
}

// NegationBuild delegates to Gt to render the inverse comparison operator.
func (lte Lte) NegationBuild(builder Builder) {
	Gt(lte).Build(builder)
}

// Like compares a column using the database's LIKE pattern syntax, not a regular
// expression. Bind the pattern as Value; % and _ retain wildcard meaning unless escaped
// using database-specific rules.
type Like Eq

// Build writes a quoted column followed by LIKE and a bound pattern. It does not escape
// wildcard characters for literal matching.
func (like Like) Build(builder Builder) {
	builder.WriteQuoted(like.Column)
	builder.WriteString(" LIKE ")
	builder.AddVar(builder, like.Value)
}

// NegationBuild writes NOT LIKE with the same bound pattern and database wildcard
// rules.
func (like Like) NegationBuild(builder Builder) {
	builder.WriteQuoted(like.Column)
	builder.WriteString(" NOT LIKE ")
	builder.AddVar(builder, like.Value)
}

// eqNil checks nil after unwrapping a non-nil driver.Valuer, then recognizes nil
// pointers. It ignores Valuer conversion errors for this predicate-shape decision.
func eqNil(value interface{}) bool {
	if valuer, ok := value.(driver.Valuer); ok && !eqNilReflect(valuer) {
		value, _ = valuer.Value()
	}

	return value == nil || eqNilReflect(value)
}

// eqNilReflect reports whether value is a nil pointer. It does not classify nil slices
// or maps as SQL NULL.
func eqNilReflect(value interface{}) bool {
	reflectValue := reflect.ValueOf(value)
	return reflectValue.Kind() == reflect.Ptr && reflectValue.IsNil()
}
