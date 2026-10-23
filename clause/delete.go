package clause

// Delete starts a DELETE statement with an optional trusted modifier. FROM and WHERE
// are separate clauses. This SQL component alone neither soft-deletes nor enforces
// GORM's missing-WHERE protection.
type Delete struct {
	// Modifier is trusted SQL appended after DELETE; database support varies.
	Modifier string
}

// Name returns DELETE, the statement registry key for this clause.
func (d Delete) Name() string {
	return "DELETE"
}

// Build writes DELETE and an optional trusted modifier. It supplies its own keyword.
func (d Delete) Build(builder Builder) {
	builder.WriteString("DELETE")

	if d.Modifier != "" {
		builder.WriteByte(' ')
		builder.WriteString(d.Modifier)
	}
}

// MergeClause clears the outer keyword to avoid duplicate DELETE and replaces the
// expression.
func (d Delete) MergeClause(clause *Clause) {
	clause.Name = ""
	clause.Expression = d
}
