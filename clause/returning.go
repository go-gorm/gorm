package clause

type Returning struct {
	Columns []Column
}

// Name where clause name
func (returning Returning) Name() string {
	return "RETURNING"
}

// Build build where clause
func (returning Returning) Build(builder Builder) {
	for idx, column := range returning.Columns {
		if idx > 0 {
			builder.WriteByte(',')
		}

		builder.WriteQuoted(column)
	}
}

// MergeClause merge order by clauses
func (returning Returning) MergeClause(clause *Clause) {
	if v, ok := clause.Expression.(Returning); ok {
		returning.Columns = append(v.Columns, returning.Columns...)
	}

	clause.Expression = returning
}
