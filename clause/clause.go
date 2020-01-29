package clause

// Builder builder interface
type BuilderInterface interface {
	Write(sql ...string) error
	WriteQuoted(field interface{}) error
	AddVar(vars ...interface{}) string
	Quote(field interface{}) string
}

// Interface clause interface
type Interface interface {
	Name() string
	Builder
}

// Builder condition builder
type Builder interface {
	Build(builder BuilderInterface)
}

// NegationBuilder negation condition builder
type NegationBuilder interface {
	NegationBuild(builder BuilderInterface)
}

// Where where clause
type Where struct {
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
