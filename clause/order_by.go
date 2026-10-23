package clause

// OrderByColumn defines one sort key. Desc reverses its direction; Reorder can discard
// earlier keys when merged into OrderBy.
type OrderByColumn struct {
	// Column is the quoted sort identifier or a deliberately raw trusted expression.
	Column Column
	// Desc emits DESC when true; false leaves the database's default ascending order.
	Desc bool
	// Reorder makes merging discard earlier keys and keep this and subsequent new keys. It
	// has no standalone SQL output.
	Reorder bool
}

// OrderBy sorts query results by Columns or a custom Expression. Repeated columns
// append unless a newly supplied OrderByColumn has Reorder set. The database determines
// collation and NULL ordering.
type OrderBy struct {
	// Columns contains ordered sort keys.
	Columns []OrderByColumn
	// Expression overrides column-list rendering with a custom trusted SQL expression.
	Expression Expression
}

// Name returns ORDER BY, the statement registry key for this clause.
func (orderBy OrderBy) Name() string {
	return "ORDER BY"
}

// Build renders Expression when non-nil; otherwise it writes quoted columns and DESC
// markers in order.
func (orderBy OrderBy) Build(builder Builder) {
	if orderBy.Expression != nil {
		orderBy.Expression.Build(builder)
	} else {
		for idx, column := range orderBy.Columns {
			if idx > 0 {
				builder.WriteByte(',')
			}

			builder.WriteQuoted(column.Column)
			if column.Desc {
				builder.WriteString(" DESC")
			}
		}
	}
}

// MergeClause appends new columns after existing ones unless a new Reorder marker is
// present; then it keeps columns from the last such marker onward. The new Expression,
// if any, takes precedence during rendering.
func (orderBy OrderBy) MergeClause(clause *Clause) {
	if v, ok := clause.Expression.(OrderBy); ok {
		for i := len(orderBy.Columns) - 1; i >= 0; i-- {
			if orderBy.Columns[i].Reorder {
				orderBy.Columns = orderBy.Columns[i:]
				clause.Expression = orderBy
				return
			}
		}

		copiedColumns := make([]OrderByColumn, len(v.Columns))
		copy(copiedColumns, v.Columns)
		orderBy.Columns = append(copiedColumns, orderBy.Columns...)
	}

	clause.Expression = orderBy
}
