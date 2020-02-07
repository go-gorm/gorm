package clause

// GroupBy group by clause
type GroupBy struct {
	Columns []Column
	Having  Where
}

// Name from clause name
func (groupBy GroupBy) Name() string {
	return "GROUP BY"
}

// Build build group by clause
func (groupBy GroupBy) Build(builder Builder) {
	for idx, column := range groupBy.Columns {
		if idx > 0 {
			builder.WriteByte(',')
		}

		builder.WriteQuoted(column)
	}

	if len(groupBy.Having.Exprs) > 0 {
		builder.Write(" HAVING ")
		groupBy.Having.Build(builder)
	}
}

// MergeClause merge group by clause
func (groupBy GroupBy) MergeClause(clause *Clause) {
	if v, ok := clause.Expression.(GroupBy); ok {
		groupBy.Columns = append(v.Columns, groupBy.Columns...)
		groupBy.Having.Exprs = append(v.Having.Exprs, groupBy.Having.Exprs...)
	}
	clause.Expression = groupBy
}
