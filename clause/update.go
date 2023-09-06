package clause

type Update struct {
	Modifier string
	Table    Table
	Joins    []Join
}

// Name update clause name
func (update Update) Name() string {
	return "UPDATE"
}

// Build build update clause
func (update Update) Build(builder Builder) {
	if update.Modifier != "" {
		builder.WriteString(update.Modifier)
		builder.WriteByte(' ')
	}

	if update.Table.Name == "" {
		builder.WriteQuoted(currentTable)
	} else {
		builder.WriteQuoted(update.Table)
	}

	for _, join := range update.Joins {
		builder.WriteByte(' ')
		join.Build(builder)
	}
}

// MergeClause merge update clause
func (update Update) MergeClause(clause *Clause) {
	if v, ok := clause.Expression.(Update); ok {
		if update.Modifier == "" {
			update.Modifier = v.Modifier
		}
		if update.Table.Name == "" {
			update.Table = v.Table
		}
	}
	clause.Expression = update
}
