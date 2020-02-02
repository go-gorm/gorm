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

// Column quote with name
type Column struct {
	Table string
	Name  string
	Alias string
	Raw   bool
}

func ToColumns(value ...interface{}) []Column {
	return nil
}

// Table quote with name
type Table struct {
	Table string
	Alias string
	Raw   bool
}
