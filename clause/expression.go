package clause

import (
	"database/sql"
	"database/sql/driver"
	"reflect"
)

// Expression expression interface
type Expression interface {
	Build(builder Builder)
}

// NegationExpressionBuilder negation expression builder
type NegationExpressionBuilder interface {
	NegationBuild(builder Builder)
}

// Expr raw expression
type Expr struct {
	SQL  string
	Vars []interface{}
}

// Build build raw expression
func (expr Expr) Build(builder Builder) {
	var (
		afterParenthesis bool
		idx              int
	)

	for _, v := range []byte(expr.SQL) {
		if v == '?' {
			if afterParenthesis {
				if _, ok := expr.Vars[idx].(driver.Valuer); ok {
					builder.AddVar(builder, expr.Vars[idx])
				} else {
					switch rv := reflect.ValueOf(expr.Vars[idx]); rv.Kind() {
					case reflect.Slice, reflect.Array:
						for i := 0; i < rv.Len(); i++ {
							if i > 0 {
								builder.WriteByte(',')
							}
							builder.AddVar(builder, rv.Index(i).Interface())
						}
					default:
						builder.AddVar(builder, expr.Vars[idx])
					}
				}
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
}

// NamedExpr raw expression for named expr
type NamedExpr struct {
	SQL  string
	Vars []interface{}
}

// Build build raw expression
func (expr NamedExpr) Build(builder Builder) {
	var (
		idx      int
		inName   bool
		namedMap = make(map[string]interface{}, len(expr.Vars))
	)

	for _, v := range expr.Vars {
		switch value := v.(type) {
		case sql.NamedArg:
			namedMap[value.Name] = value.Value
		case map[string]interface{}:
			for k, v := range value {
				namedMap[k] = v
			}
		}
	}

	name := make([]byte, 0, 10)

	for _, v := range []byte(expr.SQL) {
		if v == '@' && !inName {
			inName = true
			name = []byte{}
		} else if v == ' ' || v == ',' || v == ')' || v == '"' || v == '\'' || v == '`' {
			if inName {
				if nv, ok := namedMap[string(name)]; ok {
					builder.AddVar(builder, nv)
				} else {
					builder.WriteByte('@')
					builder.WriteString(string(name))
				}
				inName = false
			}

			builder.WriteByte(v)
		} else if v == '?' {
			builder.AddVar(builder, expr.Vars[idx])
			idx++
		} else if inName {
			name = append(name, v)
		} else {
			builder.WriteByte(v)
		}
	}

	if inName {
		builder.AddVar(builder, namedMap[string(name)])
	}
}

// IN Whether a value is within a set of values
type IN struct {
	Column interface{}
	Values []interface{}
}

func (in IN) Build(builder Builder) {
	builder.WriteQuoted(in.Column)

	switch len(in.Values) {
	case 0:
		builder.WriteString(" IN (NULL)")
	case 1:
		builder.WriteString(" = ")
		builder.AddVar(builder, in.Values...)
	default:
		builder.WriteString(" IN (")
		builder.AddVar(builder, in.Values...)
		builder.WriteByte(')')
	}
}

func (in IN) NegationBuild(builder Builder) {
	switch len(in.Values) {
	case 0:
	case 1:
		builder.WriteQuoted(in.Column)
		builder.WriteString(" <> ")
		builder.AddVar(builder, in.Values...)
	default:
		builder.WriteQuoted(in.Column)
		builder.WriteString(" NOT IN (")
		builder.AddVar(builder, in.Values...)
		builder.WriteByte(')')
	}
}

// Eq equal to for where
type Eq struct {
	Column interface{}
	Value  interface{}
}

func (eq Eq) Build(builder Builder) {
	builder.WriteQuoted(eq.Column)

	if eq.Value == nil {
		builder.WriteString(" IS NULL")
	} else {
		builder.WriteString(" = ")
		builder.AddVar(builder, eq.Value)
	}
}

func (eq Eq) NegationBuild(builder Builder) {
	Neq{eq.Column, eq.Value}.Build(builder)
}

// Neq not equal to for where
type Neq Eq

func (neq Neq) Build(builder Builder) {
	builder.WriteQuoted(neq.Column)

	if neq.Value == nil {
		builder.WriteString(" IS NOT NULL")
	} else {
		builder.WriteString(" <> ")
		builder.AddVar(builder, neq.Value)
	}
}

func (neq Neq) NegationBuild(builder Builder) {
	Eq{neq.Column, neq.Value}.Build(builder)
}

// Gt greater than for where
type Gt Eq

func (gt Gt) Build(builder Builder) {
	builder.WriteQuoted(gt.Column)
	builder.WriteString(" > ")
	builder.AddVar(builder, gt.Value)
}

func (gt Gt) NegationBuild(builder Builder) {
	Lte{gt.Column, gt.Value}.Build(builder)
}

// Gte greater than or equal to for where
type Gte Eq

func (gte Gte) Build(builder Builder) {
	builder.WriteQuoted(gte.Column)
	builder.WriteString(" >= ")
	builder.AddVar(builder, gte.Value)
}

func (gte Gte) NegationBuild(builder Builder) {
	Lt{gte.Column, gte.Value}.Build(builder)
}

// Lt less than for where
type Lt Eq

func (lt Lt) Build(builder Builder) {
	builder.WriteQuoted(lt.Column)
	builder.WriteString(" < ")
	builder.AddVar(builder, lt.Value)
}

func (lt Lt) NegationBuild(builder Builder) {
	Gte{lt.Column, lt.Value}.Build(builder)
}

// Lte less than or equal to for where
type Lte Eq

func (lte Lte) Build(builder Builder) {
	builder.WriteQuoted(lte.Column)
	builder.WriteString(" <= ")
	builder.AddVar(builder, lte.Value)
}

func (lte Lte) NegationBuild(builder Builder) {
	Gt{lte.Column, lte.Value}.Build(builder)
}

// Like whether string matches regular expression
type Like Eq

func (like Like) Build(builder Builder) {
	builder.WriteQuoted(like.Column)
	builder.WriteString(" LIKE ")
	builder.AddVar(builder, like.Value)
}

func (like Like) NegationBuild(builder Builder) {
	builder.WriteQuoted(like.Column)
	builder.WriteString(" NOT LIKE ")
	builder.AddVar(builder, like.Value)
}
