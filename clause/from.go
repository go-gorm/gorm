package clause

// From from clause
type From struct {
	Tables         []Table
	Joins          []Join
	AsOfSystemTime *AsOfSystemTime // CockroachDB specific: AS OF SYSTEM TIME
}

// Name from clause name
func (from From) Name() string {
	return "FROM"
}

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

	for _, join := range from.Joins {
		builder.WriteByte(' ')
		join.Build(builder)
	}

	// Add AS OF SYSTEM TIME clause if specified
	if from.AsOfSystemTime != nil {
		builder.WriteByte(' ')
		from.AsOfSystemTime.Build(builder)
	}
}

// MergeClause merge from clause
func (from From) MergeClause(clause *Clause) {
	clause.Expression = from
}
