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
	if len(returning.Columns) > 0 {
		for idx, column := range returning.Columns {
			if idx > 0 {
				builder.WriteByte(',')
			}

			builder.WriteQuoted(column)
		}
	} else {
		builder.WriteByte('*')
	}
}

// MergeClause merge order by clauses
func (returning Returning) MergeClause(clause *Clause) {
	if v, ok := clause.Expression.(Returning); ok && len(returning.Columns) > 0 {
		if v.Columns != nil {
			copiedColumns := make([]Column, len(v.Columns), len(v.Columns)+len(returning.Columns))
			copy(copiedColumns, v.Columns)
			returning.Columns = append(copiedColumns, returning.Columns...)
		} else {
			returning.Columns = nil
		}
	}
	clause.Expression = returning
}
