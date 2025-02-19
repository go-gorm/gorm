package clause

type OrderByColumn struct {
	Column  Column
	Desc    bool
	Reorder bool
}

func (column OrderByColumn) Build(builder Builder) {
	builder.WriteQuoted(column.Column)
	if column.Desc {
		builder.WriteString(" DESC")
	}
}

type OrderBy struct {
	Exprs []Expression
}

func (orderBy OrderBy) Name() string {
	return "ORDER BY"
}

// Build build where clause
func (orderBy OrderBy) Build(builder Builder) {
	for idx, expression := range orderBy.Exprs {
		if idx > 0 {
			builder.WriteByte(',')
		}
		expression.Build(builder)
	}
}

// MergeClause merge order by clauses
func (orderBy OrderBy) MergeClause(clause *Clause) {
	if v, ok := clause.Expression.(OrderBy); ok {
		for i := len(orderBy.Exprs) - 1; i >= 0; i-- {
			if asColumn, ok := orderBy.Exprs[i].(OrderByColumn); ok && asColumn.Reorder {
				orderBy.Exprs = orderBy.Exprs[i:]
				clause.Expression = orderBy
				return
			}
		}

		copiedColumns := make([]Expression, len(v.Exprs))
		copy(copiedColumns, v.Exprs)
		orderBy.Exprs = append(copiedColumns, orderBy.Exprs...)
	}

	clause.Expression = orderBy
}
