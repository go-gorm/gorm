package clause

// From supplies table identifiers and joins. Without Tables it renders the statement's
// current table. Repeated From clauses replace the earlier table/join selection.
type From struct {
	// Tables lists source tables; an empty list uses CurrentTable.
	Tables []Table
	// Joins lists joins rendered after the source tables in order.
	Joins []Join
}

// Name returns FROM, the statement registry key for this clause.
func (from From) Name() string {
	return "FROM"
}

// Build writes explicit Tables or the current table, then each Join separated by
// spaces.
func (from From) Build(builder Builder) {
	if len(from.Tables) > 0 {
		for idx, table := range from.Tables {
			if idx > 0 {
				builder.WriteByte(',')
			}

			builder.WriteQuoted(table)
		}
	} else {
		builder.WriteQuoted(currentTable)
	}

	for _, join := range from.Joins {
		builder.WriteByte(' ')
		join.Build(builder)
	}
}

// MergeClause replaces the existing FROM expression, including earlier tables and
// joins.
func (from From) MergeClause(clause *Clause) {
	clause.Expression = from
}
