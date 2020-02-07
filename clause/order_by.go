package clause

type OrderByColumn struct {
	Column  Column
	Desc    bool
	Reorder bool
}

type OrderBy struct {
	Columns []OrderByColumn
}

// Name where clause name
func (orderBy OrderBy) Name() string {
	return "ORDER BY"
}

// Build build where clause
func (orderBy OrderBy) Build(builder Builder) {
	for idx, column := range orderBy.Columns {
		if idx > 0 {
			builder.WriteByte(',')
		}

		builder.WriteQuoted(column.Column)
		if column.Desc {
			builder.Write(" DESC")
		}
	}
}

// MergeClause merge order by clauses
func (orderBy OrderBy) MergeClause(clause *Clause) {
	if v, ok := clause.Expression.(OrderBy); ok {
		for i := len(orderBy.Columns) - 1; i >= 0; i-- {
			if orderBy.Columns[i].Reorder {
				orderBy.Columns = orderBy.Columns[i:]
				clause.Expression = orderBy
				return
			}
		}

		orderBy.Columns = append(v.Columns, orderBy.Columns...)
	}

	clause.Expression = orderBy
}
