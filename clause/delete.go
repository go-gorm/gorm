package clause

type Delete struct {
	Modifier string
	Table    string
}

func (d Delete) Name() string {
	return "DELETE"
}

func (d Delete) Build(builder Builder) {
	builder.WriteString("DELETE")

	if d.Modifier != "" {
		builder.WriteByte(' ')
		builder.WriteString(d.Modifier)
	}
	if d.Table != "" {
		builder.WriteByte(' ')
		builder.WriteQuoted(d.Table)
	}
}

func (d Delete) MergeClause(clause *Clause) {
	clause.Name = ""
	clause.Expression = d
}
