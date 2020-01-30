package clause

// Clause
type Clause struct {
	Name                 string // WHERE
	Priority             float64
	BeforeExpressions    []Expression
	AfterNameExpressions []Expression
	AfterExpressions     []Expression
	Expression           Expression
	Builder              ClauseBuilder
}

// ClauseBuilder clause builder, allows to custmize how to build clause
type ClauseBuilder interface {
	Build(Clause, Builder)
}

// Build build clause
func (c Clause) Build(builder Builder) {
	if c.Builder != nil {
		c.Builder.Build(c, builder)
	} else {
		builders := c.BeforeExpressions
		if c.Name != "" {
			builders = append(builders, Expr{c.Name})
		}

		builders = append(builders, c.AfterNameExpressions...)
		if c.Expression != nil {
			builders = append(builders, c.Expression)
		}

		for idx, expr := range append(builders, c.AfterExpressions...) {
			if idx != 0 {
				builder.WriteByte(' ')
			}
			expr.Build(builder)
		}
	}
}

// Interface clause interface
type Interface interface {
	Name() string
	Build(Builder)
	MergeExpression(Expression)
}

type OverrideNameInterface interface {
	OverrideName() string
}

////////////////////////////////////////////////////////////////////////////////
// Predefined Clauses
////////////////////////////////////////////////////////////////////////////////

// Where where clause
type Where struct {
	AndConditions AddConditions
	ORConditions  []ORConditions
	Builders      []Expression
}

func (where Where) Name() string {
	return "WHERE"
}

func (where Where) Build(builder Builder) {
	var withConditions bool

	if len(where.AndConditions) > 0 {
		withConditions = true
		where.AndConditions.Build(builder)
	}

	if len(where.Builders) > 0 {
		for _, b := range where.Builders {
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

func (where Where) MergeExpression(expr Expression) {
	if w, ok := expr.(Where); ok {
		where.AndConditions = append(where.AndConditions, w.AndConditions...)
		where.ORConditions = append(where.ORConditions, w.ORConditions...)
		where.Builders = append(where.Builders, w.Builders...)
	} else {
		where.Builders = append(where.Builders, expr)
	}
}

// Select select attrs when querying, updating, creating
type Select struct {
	Omit bool
}

// Join join clause
type Join struct {
}

// GroupBy group by clause
type GroupBy struct {
}

// Having having clause
type Having struct {
}

// Order order clause
type Order struct {
}

// Limit limit clause
type Limit struct {
}

// Offset offset clause
type Offset struct {
}
