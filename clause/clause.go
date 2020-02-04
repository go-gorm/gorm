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

// OverrideNameInterface override name interface
type OverrideNameInterface interface {
	OverrideName() string
}

// ClauseBuilder clause builder, allows to custmize how to build clause
type ClauseBuilder interface {
	Build(Clause, Builder)
}

// Builder builder interface
type Builder interface {
	WriteByte(byte) error
	Write(sql ...string) error
	WriteQuoted(field interface{}) error
	AddVar(vars ...interface{}) string
	Quote(field interface{}) string
}
