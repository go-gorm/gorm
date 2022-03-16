package clause

import "strings"

// With Common Table Expressions
type With struct {
	Recursive  bool
	Exprs      []Expression
	Expression Expression
}

// Name with clause name
func (with With) Name() string {
	return "WITH"
}

// Build build with clause
func (with With) Build(builder Builder) {
	if with.Expression != nil {
		with.Expression.Build(builder)
		return
	}
	if len(with.Exprs) == 0 {
		return
	}

	if with.Recursive {
		builder.WriteString("RECURSIVE ")
	}
	for idx, expr := range with.Exprs {
		if idx != 0 {
			builder.WriteByte(',')
		}
		expr.Build(builder)
	}
}

// MergeClause merge with clauses
func (with With) MergeClause(clause *Clause) {
	if w, ok := clause.Expression.(With); ok {
		if !with.Recursive {
			with.Recursive = w.Recursive
		}
		if w.Expression != nil {
			with.Expression = w.Expression
			with.Exprs = nil
		} else if with.Expression == nil {
			exprs := make([]Expression, len(w.Exprs)+len(with.Exprs))
			copy(exprs, w.Exprs)
			copy(exprs[len(w.Exprs):], with.Exprs)
			with.Exprs = exprs
		}
	}
	clause.Expression = with
}

// WithExpression with expression
type WithExpression struct {
	Name    string
	Columns []string
	Expr    Expression
}

func (with WithExpression) Build(builder Builder) {
	if with.Name == "" || with.Expr == nil {
		return
	}

	builder.WriteQuoted(with.Name)

	if len(with.Columns) > 0 {
		builder.WriteByte(' ')
		builder.WriteByte('(')
		for idx, column := range with.Columns {
			if idx != 0 {
				builder.WriteByte(',')
			}
			column = strings.TrimSpace(column)
			builder.WriteQuoted(column)
		}
		builder.WriteByte(')')
	}

	builder.WriteString(" AS ")
	builder.WriteByte('(')
	with.Expr.Build(builder)
	builder.WriteByte(')')
}
