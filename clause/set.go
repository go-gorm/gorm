package clause

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
			builder.Write(builder.AddVar(assignment.Value))
		}
	} else {
		builder.WriteQuoted(PrimaryColumn)
		builder.WriteByte('=')
		builder.WriteQuoted(PrimaryColumn)
	}
}

// MergeClause merge assignments clauses
func (set Set) MergeClause(clause *Clause) {
	if v, ok := clause.Expression.(Set); ok {
		set = append(v, set...)
	}
	clause.Expression = set
}
