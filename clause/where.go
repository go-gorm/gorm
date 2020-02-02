package clause

// Where where clause
type Where struct {
	AndConditions AddConditions
	ORConditions  []ORConditions
	builders      []Expression
}

// Name where clause name
func (where Where) Name() string {
	return "WHERE"
}

// Build build where clause
func (where Where) Build(builder Builder) {
	var withConditions bool

	if len(where.AndConditions) > 0 {
		withConditions = true
		where.AndConditions.Build(builder)
	}

	if len(where.builders) > 0 {
		for _, b := range where.builders {
			if withConditions {
				builder.Write(" AND ")
			}
			withConditions = true
			b.Build(builder)
		}
	}

	var singleOrConditions []ORConditions
	for _, or := range where.ORConditions {
		if len(or) == 1 {
			if withConditions {
				builder.Write(" OR ")
				or.Build(builder)
			} else {
				singleOrConditions = append(singleOrConditions, or)
			}
		} else {
			withConditions = true
			builder.Write(" AND (")
			or.Build(builder)
			builder.WriteByte(')')
		}
	}

	for _, or := range singleOrConditions {
		if withConditions {
			builder.Write(" AND ")
			or.Build(builder)
		} else {
			withConditions = true
			or.Build(builder)
		}
	}

	if !withConditions {
		builder.Write(" FALSE")
	}

	return
}

// MergeExpression merge where clauses
func (where Where) MergeExpression(expr Expression) {
	if w, ok := expr.(Where); ok {
		where.AndConditions = append(where.AndConditions, w.AndConditions...)
		where.ORConditions = append(where.ORConditions, w.ORConditions...)
		where.builders = append(where.builders, w.builders...)
	} else {
		where.builders = append(where.builders, expr)
	}
}
