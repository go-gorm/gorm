package clause

const (
	PrimaryKey   string = "@@@priamry_key@@@"
	CurrentTable string = "@@@table@@@"
)

// Expression expression interface
type Expression interface {
	Build(builder Builder)
}

// NegationExpressionBuilder negation expression builder
type NegationExpressionBuilder interface {
	NegationBuild(builder Builder)
}

// Column quote with name
type Column struct {
	Table string
	Name  string
	Alias string
	Raw   bool
}

// Table quote with name
type Table struct {
	Table string
	Alias string
	Raw   bool
}

// Expr raw expression
type Expr struct {
	Value string
}

// Build build raw expression
func (expr Expr) Build(builder Builder) {
	builder.Write(expr.Value)
}
