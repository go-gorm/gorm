package clause

import "gorm.io/gorm/utils"

// JoinType is a trusted SQL join keyword used by Join and generic association join
// targets. Database support differs, especially for RIGHT and CROSS joins.
type JoinType string

const (
	// CrossJoin requests a CROSS JOIN; predicate support depends on database syntax.
	CrossJoin JoinType = "CROSS"
	// InnerJoin requests an INNER JOIN, excluding rows without a match.
	InnerJoin JoinType = "INNER"
	// LeftJoin requests a LEFT JOIN, retaining unmatched left-side rows.
	LeftJoin JoinType = "LEFT"
	// RightJoin requests a RIGHT JOIN on supporting databases, retaining unmatched
	// right-side rows.
	RightJoin JoinType = "RIGHT"
)

// JoinTarget describes a generic association join, optionally sourced from a subquery
// and assigned a table alias. Build it with a JoinType method or Has and pass it to the
// generic Joins API.
type JoinTarget struct {
	// Type selects the join form; an empty type uses the consuming builder's default.
	Type JoinType
	// Association is the Go model relationship path, including nested dot-separated paths
	// where supported.
	Association string
	// Subquery optionally replaces the association table source with a query expression.
	Subquery Expression
	// Table is an explicit join alias; when empty, GORM derives it from the association
	// path.
	Table string
}

// Has creates an INNER JOIN target for a named association, so parent rows without a
// matching association are excluded. It constructs metadata without executing a query.
func Has(name string) JoinTarget {
	return JoinTarget{Type: InnerJoin, Association: name}
}

// Association constructs a target for name using this join type. name is a model
// relationship path, not raw SQL.
func (jt JoinType) Association(name string) JoinTarget {
	return JoinTarget{Type: jt, Association: name}
}

// AssociationFrom constructs a target using a subquery expression as the relation
// source. Use As to choose an alias and a Joins callback to supply additional ON
// conditions.
func (jt JoinType) AssociationFrom(name string, subquery Expression) JoinTarget {
	return JoinTarget{Type: jt, Association: name, Subquery: subquery}
}

// As returns a copy of the target with name as its table alias. The original target is
// unchanged.
func (jt JoinTarget) As(name string) JoinTarget {
	jt.Table = name
	return jt
}

// Join renders a table join with an optional ON or USING condition. Expression
// overrides all structured fields. It does not perform GORM association loading by
// itself.
type Join struct {
	// Type is the trusted SQL join form before JOIN; empty emits plain JOIN.
	Type JoinType
	// Table is the quoted joined relation and optional alias.
	Table Table
	// ON contains join predicates and takes precedence over Using when non-empty.
	ON Where
	// Using lists quoted common columns used only when ON is empty.
	Using []string
	// Expression replaces structured join rendering, for example with a trusted subquery
	// join.
	Expression Expression
}

// JoinTable creates a table identifier from nested association names using GORM's
// double-underscore relation alias convention. It does not create a physical table.
func JoinTable(names ...string) Table {
	return Table{
		Name: utils.JoinNestedRelationNames(names),
	}
}

// Build renders Expression when supplied; otherwise it writes type, JOIN, table, and ON
// predicates or USING columns. ON takes precedence and all plain identifiers are quoted
// through Builder.
func (join Join) Build(builder Builder) {
	if join.Expression != nil {
		join.Expression.Build(builder)
	} else {
		if join.Type != "" {
			builder.WriteString(string(join.Type))
			builder.WriteByte(' ')
		}

		builder.WriteString("JOIN ")
		builder.WriteQuoted(join.Table)

		if len(join.ON.Exprs) > 0 {
			builder.WriteString(" ON ")
			join.ON.Build(builder)
		} else if len(join.Using) > 0 {
			builder.WriteString(" USING (")
			for idx, c := range join.Using {
				if idx > 0 {
					builder.WriteByte(',')
				}
				builder.WriteQuoted(c)
			}
			builder.WriteByte(')')
		}
	}
}
