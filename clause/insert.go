package clause

// Insert defines the target and optional trusted modifier of an INSERT. The zero table
// resolves to CurrentTable; values are supplied separately through Values or a query
// expression.
type Insert struct {
	// Table is the target table; an empty name uses CurrentTable.
	Table Table
	// Modifier is trusted SQL placed after INSERT, with dialect-specific meaning. It is not
	// a bound parameter.
	Modifier string
}

// Name returns INSERT, the statement registry key for this clause.
func (insert Insert) Name() string {
	return "INSERT"
}

// Build writes a trusted modifier, INTO, and the quoted explicit or current table. The
// outer Clause supplies INSERT.
func (insert Insert) Build(builder Builder) {
	if insert.Modifier != "" {
		builder.WriteString(insert.Modifier)
		builder.WriteByte(' ')
	}

	builder.WriteString("INTO ")
	if insert.Table.Name == "" {
		builder.WriteQuoted(currentTable)
	} else {
		builder.WriteQuoted(insert.Table)
	}
}

// MergeClause preserves the previous modifier or table when the new value omits them,
// then replaces the expression.
func (insert Insert) MergeClause(clause *Clause) {
	if v, ok := clause.Expression.(Insert); ok {
		if insert.Modifier == "" {
			insert.Modifier = v.Modifier
		}
		if insert.Table.Name == "" {
			insert.Table = v.Table
		}
	}
	clause.Expression = insert
}
