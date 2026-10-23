package clause

import (
	"strings"
)

const (
	// AndWithSpace separates SQL predicates with an AND operator and surrounding spaces.
	AndWithSpace = " AND "
	// OrWithSpace separates SQL predicates with an OR operator and surrounding spaces.
	OrWithSpace = " OR "
)

// Where is a list of predicates, normally combined with AND. Single-expression
// OrConditions introduce OR links; explicitly nested And/Or expressions control
// grouping. Repeated Where clauses append rather than replace predicates.
type Where struct {
	// Exprs holds predicates in construction order. Build may reorder the slice to avoid a
	// leading single-expression OR term; do not share it for concurrent mutation.
	Exprs []Expression
}

// Name returns WHERE, the clause registry key and rendered keyword.
func (where Where) Name() string {
	return "WHERE"
}

// Build flattens a single AndConditions group, moves a non-single-OR term to the front
// when needed, and renders predicates. Reordering can mutate the underlying expression
// slice. Nested operators and raw SQL containing AND/OR are grouped by the expression
// builder.
func (where Where) Build(builder Builder) {
	if len(where.Exprs) == 1 {
		if andCondition, ok := where.Exprs[0].(AndConditions); ok {
			where.Exprs = andCondition.Exprs
		}
	}

	// Switch position if the first query expression is a single Or condition
	for idx, expr := range where.Exprs {
		if v, ok := expr.(OrConditions); !ok || len(v.Exprs) > 1 {
			if idx != 0 {
				where.Exprs[0], where.Exprs[idx] = where.Exprs[idx], where.Exprs[0]
			}
			break
		}
	}

	buildExprs(where.Exprs, builder, AndWithSpace)
}

// buildExprs joins predicates with joinCond, except single-expression OrConditions use
// OR. It adds parentheses around raw expressions containing spaced AND/OR tokens; this
// heuristic is not a complete SQL parser.
func buildExprs(exprs []Expression, builder Builder, joinCond string) {
	wrapInParentheses := false

	for idx, expr := range exprs {
		if idx > 0 {
			if v, ok := expr.(OrConditions); ok && len(v.Exprs) == 1 {
				builder.WriteString(OrWithSpace)
			} else {
				builder.WriteString(joinCond)
			}
		}

		if len(exprs) > 1 {
			switch v := expr.(type) {
			case OrConditions:
				if len(v.Exprs) == 1 {
					if e, ok := v.Exprs[0].(Expr); ok {
						sql := strings.ToUpper(e.SQL)
						wrapInParentheses = strings.Contains(sql, AndWithSpace) || strings.Contains(sql, OrWithSpace)
					}
				}
			case AndConditions:
				if len(v.Exprs) == 1 {
					if e, ok := v.Exprs[0].(Expr); ok {
						sql := strings.ToUpper(e.SQL)
						wrapInParentheses = strings.Contains(sql, AndWithSpace) || strings.Contains(sql, OrWithSpace)
					}
				}
			case Expr:
				sql := strings.ToUpper(v.SQL)
				wrapInParentheses = strings.Contains(sql, AndWithSpace) || strings.Contains(sql, OrWithSpace)
			case NamedExpr:
				sql := strings.ToUpper(v.SQL)
				wrapInParentheses = strings.Contains(sql, AndWithSpace) || strings.Contains(sql, OrWithSpace)
			}
		}

		if wrapInParentheses {
			builder.WriteByte('(')
			expr.Build(builder)
			builder.WriteByte(')')
			wrapInParentheses = false
		} else {
			expr.Build(builder)
		}
	}
}

// MergeClause appends this clause's expressions after an existing Where, copying the
// slice container. Referenced expressions are not deep-copied.
func (where Where) MergeClause(clause *Clause) {
	if w, ok := clause.Expression.(Where); ok {
		exprs := make([]Expression, len(w.Exprs)+len(where.Exprs))
		copy(exprs, w.Exprs)
		copy(exprs[len(w.Exprs):], where.Exprs)
		where.Exprs = exprs
	}

	clause.Expression = where
}

// And combines expressions as a conjunction. It returns nil for no arguments, returns
// one non-OrConditions expression directly, and otherwise returns AndConditions. Do not
// insert a nil expression into a builder expecting Build to be callable.
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

// AndConditions groups expressions with AND and parenthesizes a multi-expression group.
// Nested single-term OrConditions retain the special connective behavior used by Where.
type AndConditions struct {
	// Exprs is the ordered group of predicates; provide non-nil expressions.
	Exprs []Expression
}

// Build renders the predicates joined by AND and wraps multiple expressions in
// parentheses using the shared boolean builder.
func (and AndConditions) Build(builder Builder) {
	if len(and.Exprs) > 1 {
		builder.WriteByte('(')
		buildExprs(and.Exprs, builder, AndWithSpace)
		builder.WriteByte(')')
	} else {
		buildExprs(and.Exprs, builder, AndWithSpace)
	}
}

// Or returns an OrConditions group, or nil for no arguments. Multiple arguments form a
// parenthesized disjunction; a single expression can also act as an OR connector in an
// enclosing Where.
func Or(exprs ...Expression) Expression {
	if len(exprs) == 0 {
		return nil
	}
	return OrConditions{Exprs: exprs}
}

// OrConditions groups predicates with OR. A single-element value has special meaning
// inside Where: it connects that term with OR to the preceding predicate.
type OrConditions struct {
	// Exprs contains the disjunctive terms; provide non-nil expressions.
	Exprs []Expression
}

// Build renders OR-separated terms and parenthesizes groups with more than one
// expression.
func (or OrConditions) Build(builder Builder) {
	if len(or.Exprs) > 1 {
		builder.WriteByte('(')
		buildExprs(or.Exprs, builder, OrWithSpace)
		builder.WriteByte(')')
	} else {
		buildExprs(or.Exprs, builder, OrWithSpace)
	}
}

// Not constructs negation, returning nil for no arguments and flattening a single
// AndConditions input. When children implement NegationExpressionBuilder, NotConditions
// can invert each child and join the results with AND; it is not a general symbolic
// boolean simplifier.
func Not(exprs ...Expression) Expression {
	if len(exprs) == 0 {
		return nil
	}
	if len(exprs) == 1 {
		if andCondition, ok := exprs[0].(AndConditions); ok {
			exprs = andCondition.Exprs
		}
	}
	return NotConditions{Exprs: exprs}
}

// NotConditions negates expressions using their NegationBuild implementation where
// available. If any term supports that interface, each term is individually inverted
// and joined with AND; otherwise it emits a NOT wrapper around the group. Explicitly
// group complex logic instead of assuming automatic De Morgan transformations.
type NotConditions struct {
	// Exprs contains the expressions to negate; their implementation determines specialized
	// inverse-operator behavior.
	Exprs []Expression
}

// Build uses inverse operators for NegationExpressionBuilder terms and NOT wrappers for
// other terms. With no inverse builders it wraps the combined expression. Parentheses
// follow the shared raw-expression grouping heuristics.
func (not NotConditions) Build(builder Builder) {
	anyNegationBuilder := false
	for _, c := range not.Exprs {
		if _, ok := c.(NegationExpressionBuilder); ok {
			anyNegationBuilder = true
			break
		}
	}

	if anyNegationBuilder {
		if len(not.Exprs) > 1 {
			builder.WriteByte('(')
		}

		for idx, c := range not.Exprs {
			if idx > 0 {
				builder.WriteString(AndWithSpace)
			}

			if negationBuilder, ok := c.(NegationExpressionBuilder); ok {
				negationBuilder.NegationBuild(builder)
			} else {
				builder.WriteString("NOT ")
				e, wrapInParentheses := c.(Expr)
				if wrapInParentheses {
					sql := strings.ToUpper(e.SQL)
					if wrapInParentheses = strings.Contains(sql, AndWithSpace) || strings.Contains(sql, OrWithSpace); wrapInParentheses {
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
	} else {
		builder.WriteString("NOT ")
		if len(not.Exprs) > 1 {
			builder.WriteByte('(')
		}

		for idx, c := range not.Exprs {
			if idx > 0 {
				switch c.(type) {
				case OrConditions:
					builder.WriteString(OrWithSpace)
				default:
					builder.WriteString(AndWithSpace)
				}
			}

			e, wrapInParentheses := c.(Expr)
			if wrapInParentheses {
				sql := strings.ToUpper(e.SQL)
				if wrapInParentheses = strings.Contains(sql, AndWithSpace) || strings.Contains(sql, OrWithSpace); wrapInParentheses {
					builder.WriteByte('(')
				}
			}

			c.Build(builder)

			if wrapInParentheses {
				builder.WriteByte(')')
			}
		}

		if len(not.Exprs) > 1 {
			builder.WriteByte(')')
		}
	}
}
