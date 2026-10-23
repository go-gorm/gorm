package clause

// OnConflict configures database-side insert conflict handling. Choose DoNothing or
// DoUpdates, and specify a supported target through Columns/TargetWhere or
// OnConstraint. UpdateAll is expanded by GORM create callbacks using model metadata.
// Driver overrides and database constraints determine actual semantics.
type OnConflict struct {
	// Columns identifies conflict-target columns, where supported.
	Columns []Column
	// Where filters rows for the update action, distinct from TargetWhere. Do not combine
	// it with an action unsupported by the database.
	Where Where
	// TargetWhere adds the conflict-target predicate, such as one identifying a partial
	// unique index.
	TargetWhere Where
	// OnConstraint names the target constraint and takes precedence over
	// Columns/TargetWhere. It is emitted verbatim, so supply trusted application text.
	OnConstraint string
	// DoNothing chooses DO NOTHING instead of an update action.
	DoNothing bool
	// DoUpdates contains assignments for the conflict update action. Use AssignmentColumns
	// to refer to incoming excluded values where supported.
	DoUpdates Set
	// UpdateAll asks GORM create callbacks to derive update assignments from writable model
	// columns, subject to primary-key/default/timestamp rules. It has no standalone effect
	// in Build.
	UpdateAll bool
}

// Name returns ON CONFLICT, the statement registry key for this clause.
func (OnConflict) Name() string {
	return "ON CONFLICT"
}

// Build renders a constraint or column target, optional target predicate, and DO
// NOTHING or DO UPDATE SET. Where is emitted after the action; callers must choose a
// combination valid for their database. UpdateAll is not expanded by this method.
func (onConflict OnConflict) Build(builder Builder) {
	if onConflict.OnConstraint != "" {
		builder.WriteString("ON CONSTRAINT ")
		builder.WriteString(onConflict.OnConstraint)
		builder.WriteByte(' ')
	} else {
		if len(onConflict.Columns) > 0 {
			builder.WriteByte('(')
			for idx, column := range onConflict.Columns {
				if idx > 0 {
					builder.WriteByte(',')
				}
				builder.WriteQuoted(column)
			}
			builder.WriteString(`) `)
		}

		if len(onConflict.TargetWhere.Exprs) > 0 {
			builder.WriteString(" WHERE ")
			onConflict.TargetWhere.Build(builder)
			builder.WriteByte(' ')
		}
	}

	if onConflict.DoNothing {
		builder.WriteString("DO NOTHING")
	} else {
		builder.WriteString("DO UPDATE SET ")
		onConflict.DoUpdates.Build(builder)
	}

	if len(onConflict.Where.Exprs) > 0 {
		builder.WriteString(" WHERE ")
		onConflict.Where.Build(builder)
		builder.WriteByte(' ')
	}
}

// MergeClause replaces the entire earlier conflict clause rather than appending
// assignments or targets.
func (onConflict OnConflict) MergeClause(clause *Clause) {
	clause.Expression = onConflict
}
