package clause

type OrderBy struct {
	Column  Column
	Desc    bool
	Reorder bool
}

type OrderByClause struct {
	Columns []OrderBy
}

// Name where clause name
func (orderBy OrderByClause) Name() string {
	return "ORDER BY"
}

// Build build where clause
func (orderBy OrderByClause) Build(builder Builder) {
	for i := len(orderBy.Columns) - 1; i >= 0; i-- {
		builder.WriteQuoted(orderBy.Columns[i].Column)

		if orderBy.Columns[i].Desc {
			builder.Write(" DESC")
		}

		if orderBy.Columns[i].Reorder {
			break
		}
	}
}

// MergeExpression merge order by clauses
func (orderBy OrderByClause) MergeExpression(expr Expression) {
	if v, ok := expr.(OrderByClause); ok {
		orderBy.Columns = append(v.Columns, orderBy.Columns...)
	}
}
