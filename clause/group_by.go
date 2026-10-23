package clause

// GroupBy combines grouping columns and HAVING predicates. Repeated clauses append both
// lists. With no grouping columns, merging suppresses the GROUP BY keyword while still
// allowing a HAVING body.
type GroupBy struct {
	// Columns holds group identifiers or explicitly raw expressions.
	Columns []Column
	// Having contains post-group predicates rendered with Where grouping rules.
	Having []Expression
}

// Name returns GROUP BY, the statement registry key for this clause.
func (groupBy GroupBy) Name() string {
	return "GROUP BY"
}

// Build writes quoted grouping columns followed by HAVING predicates when present.
func (groupBy GroupBy) Build(builder Builder) {
	for idx, column := range groupBy.Columns {
		if idx > 0 {
			builder.WriteByte(',')
		}

		builder.WriteQuoted(column)
	}

	if len(groupBy.Having) > 0 {
		builder.WriteString(" HAVING ")
		Where{Exprs: groupBy.Having}.Build(builder)
	}
}

// MergeClause appends existing grouping columns and HAVING expressions ahead of the new
// ones and suppresses the keyword when no columns exist.
func (groupBy GroupBy) MergeClause(clause *Clause) {
	if v, ok := clause.Expression.(GroupBy); ok {
		copiedColumns := make([]Column, len(v.Columns))
		copy(copiedColumns, v.Columns)
		groupBy.Columns = append(copiedColumns, groupBy.Columns...)

		copiedHaving := make([]Expression, len(v.Having))
		copy(copiedHaving, v.Having)
		groupBy.Having = append(copiedHaving, groupBy.Having...)
	}
	clause.Expression = groupBy

	if len(groupBy.Columns) == 0 {
		clause.Name = ""
	} else {
		clause.Name = groupBy.Name()
	}
}
