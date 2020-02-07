package clause

type Values struct {
	Columns []Column
	Values  [][]interface{}
}

// Name from clause name
func (Values) Name() string {
	return ""
}

// Build build from clause
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

		builder.Write(" VALUES ")

		for idx, value := range values.Values {
			if idx > 0 {
				builder.WriteByte(',')
			}

			builder.WriteByte('(')
			builder.Write(builder.AddVar(value...))
			builder.WriteByte(')')
		}
	} else {
		builder.Write("DEFAULT VALUES")
	}
}

// MergeClause merge values clauses
func (values Values) MergeClause(clause *Clause) {
	if v, ok := clause.Expression.(Values); ok {
		values.Values = append(v.Values, values.Values...)
	}
	clause.Expression = values
}
