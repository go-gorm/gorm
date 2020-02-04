package clause

// From from clause
type From struct {
	Tables []Table
}

// Name from clause name
func (From) Name() string {
	return "FROM"
}

var currentTable = Table{Table: CurrentTable}

// Build build from clause
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
}

// MergeExpression merge order by clauses
func (from From) MergeExpression(expr Expression) {
	if v, ok := expr.(From); ok {
		from.Tables = append(v.Tables, from.Tables...)
	}
}
