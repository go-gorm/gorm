package clause

import (
	"strings"
)

// Where where clause
type Where struct {
	Exprs []Expression
}

// Name where clause name
func (where Where) Name() string {
	return "WHERE"
}

// Build build where clause
func (where Where) Build(builder Builder) {
	// Switch position if the first query expression is a single Or condition
	for idx, expr := range where.Exprs {
		if v, ok := expr.(OrConditions); !ok || len(v.Exprs) > 1 {
			if idx != 0 {
				where.Exprs[0], where.Exprs[idx] = where.Exprs[idx], where.Exprs[0]
			}
			break
		}
	}

	buildExprs(where.Exprs, builder, " AND ")
}

func buildExprs(exprs []Expression, builder Builder, joinCond string) {
	wrapInParentheses := false

	for idx, expr := range exprs {
		if idx > 0 {
			if v, ok := expr.(OrConditions); ok && len(v.Exprs) == 1 {
				builder.WriteString(" OR ")
			} else {
				builder.WriteString(joinCond)
			}
		}

		if len(exprs) > 1 {
			switch v := expr.(type) {
			case OrConditions:
				if len(v.Exprs) == 1 {
					if e, ok := v.Exprs[0].(Expr); ok {
						sql := strings.ToLower(e.SQL)
						wrapInParentheses = strings.Contains(sql, "and") || strings.Contains(sql, "or")
					}
				}
			case AndConditions:
				if len(v.Exprs) == 1 {
					if e, ok := v.Exprs[0].(Expr); ok {
						sql := strings.ToLower(e.SQL)
						wrapInParentheses = strings.Contains(sql, "and") || strings.Contains(sql, "or")
					}
				}
			case Expr:
				sql := strings.ToLower(v.SQL)
				wrapInParentheses = strings.Contains(sql, "and") || strings.Contains(sql, "or")
			case NamedExpr:
				sql := strings.ToLower(v.SQL)
				wrapInParentheses = strings.Contains(sql, "and") || strings.Contains(sql, "or")
			}
		}

		if wrapInParentheses {
			builder.WriteString(`(`)
			expr.Build(builder)
			builder.WriteString(`)`)
			wrapInParentheses = false
		} else {
			expr.Build(builder)
		}
	}
}

// MergeClause merge where clauses
func (where Where) MergeClause(clause *Clause) {
	if w, ok := clause.Expression.(Where); ok {
		exprs := make([]Expression, len(w.Exprs)+len(where.Exprs))
		copy(exprs, w.Exprs)
		copy(exprs[len(w.Exprs):], where.Exprs)
		where.Exprs = exprs
	}

	clause.Expression = where
}

func And(exprs ...Expression) Expression {
	if len(exprs) == 0 {
		return nil
	}

	if len(exprs) == 1 {
		if _, ok := exprs[0].(OrConditions); !ok {
			return exprs[0]
		}
	}

	return AndConditions{Exprs: exprs}
}

type AndConditions struct {
	Exprs []Expression
}

func (and AndConditions) Build(builder Builder) {
	if len(and.Exprs) > 1 {
		builder.WriteByte('(')
		buildExprs(and.Exprs, builder, " AND ")
		builder.WriteByte(')')
	} else {
		buildExprs(and.Exprs, builder, " AND ")
	}
}

func Or(exprs ...Expression) Expression {
	if len(exprs) == 0 {
		return nil
	}
	return OrConditions{Exprs: exprs}
}

type OrConditions struct {
	Exprs []Expression
}

func (or OrConditions) Build(builder Builder) {
	if len(or.Exprs) > 1 {
		builder.WriteByte('(')
		buildExprs(or.Exprs, builder, " OR ")
		builder.WriteByte(')')
	} else {
		buildExprs(or.Exprs, builder, " OR ")
	}
}

func Not(exprs ...Expression) Expression {
	if len(exprs) == 0 {
		return nil
	}
	return NotConditions{Exprs: exprs}
}

type NotConditions struct {
	Exprs []Expression
}

func (not NotConditions) Build(builder Builder) {
	if len(not.Exprs) > 1 {
		builder.WriteByte('(')
	}

	for idx, c := range not.Exprs {
		if idx > 0 {
			builder.WriteString(" AND ")
		}

		if negationBuilder, ok := c.(NegationExpressionBuilder); ok {
			negationBuilder.NegationBuild(builder)
		} else {
			builder.WriteString("NOT ")
			e, wrapInParentheses := c.(Expr)
			if wrapInParentheses {
				sql := strings.ToLower(e.SQL)
				if wrapInParentheses = strings.Contains(sql, "and") || strings.Contains(sql, "or"); wrapInParentheses {
					builder.WriteByte('(')
				}
			}

			c.Build(builder)

			if wrapInParentheses {
				builder.WriteByte(')')
			}
		}
	}

	if len(not.Exprs) > 1 {
		builder.WriteByte(')')
	}
}
