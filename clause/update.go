package clause

// Update selects the target table and optional trusted UPDATE modifier. SET and WHERE
// are separate clauses; this builder alone does not enforce a missing-WHERE guard.
type Update struct {
	// Modifier is trusted database-specific syntax after UPDATE, not a bound data value.
	Modifier string
	// Table is the target table; an empty name resolves to CurrentTable.
	Table Table
}

// Name returns UPDATE, the statement registry key for this clause.
func (update Update) Name() string {
	return "UPDATE"
}

// Build writes the trusted modifier and quoted target table, defaulting to
// CurrentTable. The outer Clause supplies UPDATE.
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
}

// MergeClause keeps an existing modifier/table when the new value leaves it empty, then
// stores the resulting expression.
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
