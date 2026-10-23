package clause

// Values supplies insert columns and rows. Row values must align with Columns. No
// columns renders DEFAULT VALUES; this builder does not validate row widths or database
// bind-parameter limits.
type Values struct {
	// Columns defines the ordered insert columns.
	Columns []Column
	// Values contains rows in Columns order. Keep each row width consistent; values are
	// bound through Builder.AddVar.
	Values [][]interface{}
}

// Name returns VALUES, the statement registry key for this clause.
func (Values) Name() string {
	return "VALUES"
}

// Build writes quoted columns and parameterized rows, or DEFAULT VALUES when Columns is
// empty. It emits its own complete syntax.
func (values Values) Build(builder Builder) {
	if len(values.Columns) > 0 {
		builder.WriteByte('(')
		for idx, column := range values.Columns {
			if idx > 0 {
				builder.WriteByte(',')
			}
			builder.WriteQuoted(column)
		}
		builder.WriteByte(')')

		builder.WriteString(" VALUES ")

		for idx, value := range values.Values {
			if idx > 0 {
				builder.WriteByte(',')
			}

			builder.WriteByte('(')
			builder.AddVar(builder, value...)
			builder.WriteByte(')')
		}
	} else {
		builder.WriteString("DEFAULT VALUES")
	}
}

// MergeClause replaces prior values and clears the outer keyword because Build emits
// VALUES itself.
func (values Values) MergeClause(clause *Clause) {
	clause.Name = ""
	clause.Expression = values
}
