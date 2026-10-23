package clause

// Returning requests fields from a write on databases supporting returned rows. No
// Columns means *. It controls SQL projection; scan behavior and support are determined
// by write callbacks and the dialector.
type Returning struct {
	// Columns specifies returned columns; nil or empty requests all columns. Nil also
	// preserves wildcard semantics when a later clause adds columns.
	Columns []Column
}

// Name returns RETURNING, the statement registry key for this clause.
func (returning Returning) Name() string {
	return "RETURNING"
}

// Build writes quoted Columns or * when the selection is empty.
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

// MergeClause combines explicit column selections, but an existing nil column list
// keeps wildcard selection. A new empty selection replaces the result with a wildcard.
func (returning Returning) MergeClause(clause *Clause) {
	if v, ok := clause.Expression.(Returning); ok && len(returning.Columns) > 0 {
		if v.Columns != nil {
			returning.Columns = append(v.Columns, returning.Columns...)
		} else {
			returning.Columns = nil
		}
	}
	clause.Expression = returning
}
