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
			builder.WriteByte('(')
			if idx > 0 {
				builder.WriteByte(',')
			}

			builder.Write(builder.AddVar(value...))
			builder.WriteByte(')')
		}
	} else {
		builder.Write("DEFAULT VALUES")
	}
}
