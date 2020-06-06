package clause

import "sort"

type Set []Assignment

type Assignment struct {
	Column Column
	Value  interface{}
}

func (set Set) Name() string {
	return "SET"
}

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
		builder.WriteQuoted(PrimaryColumn)
		builder.WriteByte('=')
		builder.WriteQuoted(PrimaryColumn)
	}
}

// MergeClause merge assignments clauses
func (set Set) MergeClause(clause *Clause) {
	clause.Expression = set
}

func Assignments(values map[string]interface{}) Set {
	var keys []string
	var assignments []Assignment

	for key := range values {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		assignments = append(assignments, Assignment{
			Column: Column{Table: CurrentTable, Name: key},
			Value:  values[key],
		})
	}
	return assignments
}
