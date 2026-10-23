package clause

// Interface is a named SQL clause that can be merged into a GORM statement. Name
// selects its registry slot; MergeClause defines how repeated clauses combine; Build
// writes the body. Dialectors can override rendering by clause name.
type Interface interface {
	// Name returns the statement registry key, usually the SQL keyword. MergeClause may
	// clear the rendered Clause.Name when Build includes its own keyword.
	Name() string
	// Build appends the clause body to builder. Bind data through AddVar, quote identifiers
	// through WriteQuoted, and report failures through AddError.
	Build(Builder)
	// MergeClause combines this value with an existing registry entry. It must set the
	// resulting Expression and preserve or replace existing state according to the clause's
	// contract.
	MergeClause(*Clause)
}

// ClauseBuilder overrides rendering for a registered clause. It receives the merged
// Clause and writes SQL/variables through Builder; it does not execute the statement.
type ClauseBuilder func(Clause, Builder)

// Writer is the byte/string sink used by SQL builders. Text written directly is trusted
// SQL; identifier quoting and parameter binding are separate Builder operations.
type Writer interface {
	// WriteByte appends a SQL byte and reports sink errors.
	WriteByte(byte) error
	// WriteString appends SQL text, returning the byte count and sink error.
	WriteString(string) (int, error)
}

// Builder combines SQL output, identifier quoting, variable binding, and error
// recording. gorm.Statement implements it; clause implementations should use AddVar
// instead of interpolating values.
type Builder interface {
	Writer
	// WriteQuoted renders an identifier or structured Column/Table using the dialect. Raw
	// flags explicitly bypass quoting.
	WriteQuoted(field interface{})
	// AddVar writes value placeholders or nested expressions to the supplied Writer and
	// records bound values on the builder.
	AddVar(Writer, ...interface{})
	// AddError records a construction error and returns the builder's accumulated error.
	AddError(error) error
}

// Clause is a merged registry entry with optional expressions before its name, after
// its name, and after its body. A custom Builder overrides the entire default rendering
// path. A nil Expression renders nothing unless Builder is set.
type Clause struct {
	// Name is the rendered keyword, such as WHERE; an empty name lets Expression render its
	// own syntax.
	Name string // WHERE
	// BeforeExpression renders before the keyword when the main Expression is non-nil.
	BeforeExpression Expression
	// AfterNameExpression renders between the keyword and main Expression.
	AfterNameExpression Expression
	// AfterExpression renders after the main Expression, separated by a space.
	AfterExpression Expression
	// Expression is the clause body produced by MergeClause.
	Expression Expression
	// Builder optionally replaces default rendering, including the name and surrounding
	// expressions.
	Builder ClauseBuilder
}

// Build invokes the custom Builder when set; otherwise it renders non-nil Expression
// with the name and optional surrounding expressions separated by spaces. It appends
// output and does not execute SQL.
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

const (
	// PrimaryKey is a symbolic column name resolved by gorm.Statement to the model's
	// prioritized key, or its first accessible column when no key is prioritized. It is not
	// a literal database identifier.
	PrimaryKey string = "~~~py~~~" // primary key
	// CurrentTable is a symbolic table name resolved from the current statement's model or
	// table override. Use it for expressions that must follow table naming and aliases.
	CurrentTable string = "~~~ct~~~" // current table
	// Associations is the symbolic selector for all direct model associations in GORM
	// Select, Omit, and Preload. It does not recursively include every nested association.
	Associations string = "~~~as~~~" // associations
)

var (
	// currentTable is the default symbolic table used by clause builders when no explicit
	// table is supplied.
	currentTable = Table{Name: CurrentTable}
	// PrimaryColumn refers to the current model's primary-key column. Copy it when
	// customizing an expression rather than mutating this shared package variable.
	PrimaryColumn = Column{Table: CurrentTable, Name: PrimaryKey}
)

// Column describes a dialect-quoted, optionally table-qualified SQL identifier. Name
// can be PrimaryKey and Table can be CurrentTable for model-aware resolution. Raw
// bypasses quoting and must only contain trusted SQL.
type Column struct {
	// Table optionally qualifies the column; CurrentTable resolves through statement
	// metadata.
	Table string
	// Name is the column identifier, or PrimaryKey for model-key resolution.
	Name string
	// Alias adds an AS alias when the column is rendered.
	Alias string
	// Raw bypasses dialect quoting of identifier components. Never set it for unvalidated
	// user input.
	Raw bool
}

// Table describes a quoted table identifier and optional alias. CurrentTable resolves
// to the active model/table expression; Raw bypasses quoting.
type Table struct {
	// Name identifies the table, or CurrentTable for statement-dependent resolution.
	Name string
	// Alias adds a table alias after the identifier.
	Alias string
	// Raw renders table and alias text without identifier quoting; only trusted SQL belongs
	// here.
	Raw bool
}
