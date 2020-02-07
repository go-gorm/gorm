package clause

type Delete struct {
	Modifier string
}

func (d Delete) Name() string {
	return "DELETE"
}

func (d Delete) Build(builder Builder) {
	builder.Write("DELETE")

	if d.Modifier != "" {
		builder.WriteByte(' ')
		builder.Write(d.Modifier)
	}
}

func (d Delete) MergeClause(clause *Clause) {
	clause.Name = ""
	clause.Expression = d
}
