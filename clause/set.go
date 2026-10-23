package clause

import "sort"

// Set is an ordered list of column assignments for UPDATE or conflict handling. Values
// are bound or rendered as expressions. An empty Set renders primary_key=primary_key as
// a no-op assignment; it is not an empty SQL body.
type Set []Assignment

// Assignment associates a quoted Column with a value or SQL expression. It also
// implements Assigner for the generic Set API.
type Assignment struct {
	// Column is the target column identifier.
	Column Column
	// Value is bound as data unless it is a nested expression or structured SQL value.
	Value interface{}
}

// Assigner supplies column assignments for the generic Set API. Association operations
// additionally implement AssociationAssigner so they can be executed separately.
type Assigner interface {
	// Assignments returns column/value pairs; the caller may retain their referenced values
	// until execution.
	Assignments() []Assignment
}

// Name returns SET, the assignment clause registry key.
func (set Set) Name() string {
	return "SET"
}

// Build writes comma-separated quoted-column assignments with bound values. An empty
// set assigns the resolved primary-key column to itself.
func (set Set) Build(builder Builder) {
	if len(set) > 0 {
		for idx, assignment := range set {
			if idx > 0 {
				builder.WriteByte(',')
			}
			builder.WriteQuoted(assignment.Column)
			builder.WriteByte('=')
			builder.AddVar(builder, assignment.Value)
		}
	} else {
		builder.WriteQuoted(Column{Name: PrimaryKey})
		builder.WriteByte('=')
		builder.WriteQuoted(Column{Name: PrimaryKey})
	}
}

// MergeClause replaces prior assignments with a copied slice. Values inside assignments
// remain shared references.
func (set Set) MergeClause(clause *Clause) {
	copiedAssignments := make([]Assignment, len(set))
	copy(copiedAssignments, set)
	clause.Expression = Set(copiedAssignments)
}

// Assignments exposes the set as an assignment slice without copying its backing array.
func (set Set) Assignments() []Assignment { return []Assignment(set) }

// Assignments converts a string-keyed value map into a Set sorted by column name for
// deterministic SQL. Values remain referenced; use expressions explicitly for SQL
// calculations.
func Assignments(values map[string]interface{}) Set {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	assignments := make([]Assignment, len(keys))
	for idx, key := range keys {
		assignments[idx] = Assignment{Column: Column{Name: key}, Value: values[key]}
	}
	return assignments
}

// AssignmentColumns builds assignments from named columns to their incoming excluded
// counterparts for conflict updates. The dialector may translate this representation to
// its native upsert syntax.
func AssignmentColumns(values []string) Set {
	assignments := make([]Assignment, len(values))
	for idx, value := range values {
		assignments[idx] = Assignment{Column: Column{Name: value}, Value: Column{Table: "excluded", Name: value}}
	}
	return assignments
}

// Assignments returns a one-element slice containing this column assignment.
func (a Assignment) Assignments() []Assignment { return []Assignment{a} }
