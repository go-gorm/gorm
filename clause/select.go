package clause

// Select defines the query projection. Columns are quoted; an empty list renders *. A
// supplied Expression takes precedence during MergeClause. Distinct is meaningful with
// columns or an Expr selection.
type Select struct {
	// Distinct requests distinct projected rows; custom non-Expr expressions must express
	// DISTINCT themselves.
	Distinct bool
	// Columns contains projected column identifiers; an empty slice defaults to *.
	Columns []Column
	// Expression optionally replaces the column-list projection during merging.
	Expression Expression
}

// Name returns SELECT, the statement registry key for this clause.
func (s Select) Name() string {
	return "SELECT"
}

// Build writes quoted Columns, optionally prefixed with DISTINCT, or * for no columns.
// Expression is selected during MergeClause rather than by this method.
func (s Select) Build(builder Builder) {
	if len(s.Columns) > 0 {
		if s.Distinct {
			builder.WriteString("DISTINCT ")
		}

		for idx, column := range s.Columns {
			if idx > 0 {
				builder.WriteByte(',')
			}
			builder.WriteQuoted(column)
		}
	} else {
		builder.WriteByte('*')
	}
}

// MergeClause replaces the existing selection. A non-nil Expression becomes the clause
// body; Distinct prefixes it only when its concrete type is Expr.
func (s Select) MergeClause(clause *Clause) {
	if s.Expression != nil {
		if s.Distinct {
			if expr, ok := s.Expression.(Expr); ok {
				expr.SQL = "DISTINCT " + expr.SQL
				clause.Expression = expr
				return
			}
		}

		clause.Expression = s.Expression
	} else {
		clause.Expression = s
	}
}

// CommaExpression renders an ordered expression list separated by commas. It is useful
// for projections and partition keys; each child controls its own quoting and bindings.
type CommaExpression struct {
	// Exprs contains non-nil expressions in output order.
	Exprs []Expression
}

// Build renders each expression separated by comma and space, with no surrounding
// parentheses.
func (comma CommaExpression) Build(builder Builder) {
	for idx, expr := range comma.Exprs {
		if idx > 0 {
			_, _ = builder.WriteString(", ")
		}
		expr.Build(builder)
	}
}
