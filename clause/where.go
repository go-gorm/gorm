package clause

// Where where clause
type Where struct {
	Exprs []Expression
}

// Name where clause name
func (where Where) Name() string {
	return "WHERE"
}

// Build build where clause
func (where Where) Build(builder Builder) {
	// Switch position if the first query expression is a single Or condition
	for idx, expr := range where.Exprs {
		if v, ok := expr.(OrConditions); (!ok && expr != nil) || len(v.Exprs) > 1 {
			if idx != 0 {
				where.Exprs[0], where.Exprs[idx] = where.Exprs[idx], where.Exprs[0]
			}
			break
		}
	}

	for idx, expr := range where.Exprs {
		if expr != nil {
			if idx > 0 {
				if v, ok := expr.(OrConditions); ok && len(v.Exprs) == 1 {
					builder.Write(" OR ")
				} else {
					builder.Write(" AND ")
				}
			}

			expr.Build(builder)
		}
	}

	return
}

// MergeClause merge where clauses
func (where Where) MergeClause(clause *Clause) {
	if w, ok := clause.Expression.(Where); ok {
		where.Exprs = append(w.Exprs, where.Exprs...)
	}

	clause.Expression = where
}

func And(exprs ...Expression) Expression {
	if len(exprs) == 0 {
		return nil
	}
	return AndConditions{Exprs: exprs}
}

type AndConditions struct {
	Exprs []Expression
}

func (and AndConditions) Build(builder Builder) {
	if len(and.Exprs) > 1 {
		builder.WriteByte('(')
	}
	for idx, c := range and.Exprs {
		if idx > 0 {
			builder.Write(" AND ")
		}
		c.Build(builder)
	}
	if len(and.Exprs) > 1 {
		builder.WriteByte(')')
	}
}

func Or(exprs ...Expression) Expression {
	if len(exprs) == 0 {
		return nil
	}
	return OrConditions{Exprs: exprs}
}

type OrConditions struct {
	Exprs []Expression
}

func (or OrConditions) Build(builder Builder) {
	if len(or.Exprs) > 1 {
		builder.WriteByte('(')
	}
	for idx, c := range or.Exprs {
		if idx > 0 {
			builder.Write(" OR ")
		}
		c.Build(builder)
	}
	if len(or.Exprs) > 1 {
		builder.WriteByte(')')
	}
}

func Not(exprs ...Expression) Expression {
	if len(exprs) == 0 {
		return nil
	}
	return NotConditions{Exprs: exprs}
}

type NotConditions struct {
	Exprs []Expression
}

func (not NotConditions) Build(builder Builder) {
	if len(not.Exprs) > 1 {
		builder.WriteByte('(')
	}
	for idx, c := range not.Exprs {
		if idx > 0 {
			builder.Write(" AND ")
		}

		if negationBuilder, ok := c.(NegationExpressionBuilder); ok {
			negationBuilder.NegationBuild(builder)
		} else {
			builder.Write(" NOT ")
			c.Build(builder)
		}
	}
	if len(not.Exprs) > 1 {
		builder.WriteByte(')')
	}
}
