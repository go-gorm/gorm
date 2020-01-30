package clause

// Expression expression interface
type Expression interface {
	Build(builder Builder)
}

// NegationExpressionBuilder negation expression builder
type NegationExpressionBuilder interface {
	NegationBuild(builder Builder)
}

// Builder builder interface
type Builder interface {
	WriteByte(byte) error
	Write(sql ...string) error
	WriteQuoted(field interface{}) error
	AddVar(vars ...interface{}) string
	Quote(field interface{}) string
}

// Expr raw expression
type Expr struct {
	Value string
}

// Build build raw expression
func (expr Expr) Build(builder Builder) {
	builder.Write(expr.Value)
}
