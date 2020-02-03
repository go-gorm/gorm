package clause

type Insert struct {
	Table    Table
	Priority string
}

// Name insert clause name
func (insert Insert) Name() string {
	return "INSERT"
}

// Build build insert clause
func (insert Insert) Build(builder Builder) {
	if insert.Priority != "" {
		builder.Write(insert.Priority)
		builder.WriteByte(' ')
	}

	builder.Write("INTO ")
	builder.WriteQuoted(insert.Table)
}

// MergeExpression merge insert clauses
func (insert Insert) MergeExpression(expr Expression) {
	if v, ok := expr.(Insert); ok {
		if insert.Priority == "" {
			insert.Priority = v.Priority
		}
		if insert.Table.Table == "" {
			insert.Table = v.Table
		}
	}
}
