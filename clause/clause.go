package clause

// Interface is a clause interface.
type Interface interface {
	Name() string
	Build(Builder)
	MergeClause(*Clause)
}

// ClauseBuilder is a clause builder that allows customization of how to build a clause.
type ClauseBuilder func(Clause, Builder)

// Writer is an interface for writing operations.
type Writer interface {
	WriteByte(byte) error
	WriteString(string) (int, error)
}

// Builder is a builder interface.
type Builder interface {
	Writer
	WriteQuoted(field interface{})
	AddVar(Writer, ...interface{})
	AddError(error) error
}

// Clause represents a query clause.
type Clause struct {
	Name                string
	BeforeExpression    Expression
	AfterNameExpression Expression
	AfterExpression     Expression
	Expression          Expression
	Builder             ClauseBuilder
}

// Build builds the clause.
func (c Clause) Build(builder Builder) {
	if c.Builder != nil {
		c.Builder(c, builder)
	} else if c.Expression != nil {
		if c.BeforeExpression != nil {
			c.BeforeExpression.Build(builder)
			builder.WriteByte(' ')
		}

		if c.Name != "" {
			builder.WriteString(c.Name)
			builder.WriteByte(' ')
		}

		if c.AfterNameExpression != nil {
			c.AfterNameExpression.Build(builder)
			builder.WriteByte(' ')
		}

		c.Expression.Build(builder)

		if c.AfterExpression != nil {
			builder.WriteByte(' ')
			c.AfterExpression.Build(builder)
		}
	}
}

// Constants for special names.
const (
	PrimaryKey   = "~~~py~~~" // Primary key
	CurrentTable = "~~~ct~~~" // Current table
	Associations = "~~~as~~~" // Associations
)

// Predefined instances.
var (
	CurrentTableInstance = Table{Name: CurrentTable}
	PrimaryColumn       = Column{Table: CurrentTable, Name: PrimaryKey}
)

// Column represents a column with optional table and alias.
type Column struct {
	Table string
	Name  string
	Alias string
	Raw   bool
}

// Table represents a table with optional alias.
type Table struct {
	Name  string
	Alias string
	Raw   bool
}
