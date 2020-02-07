package clause

// Select select attrs when querying, updating, creating
type Select struct {
	Columns []Column
	Omits   []Column
}

func (s Select) Name() string {
	return "SELECT"
}

func (s Select) Build(builder Builder) {
	if len(s.Columns) > 0 {
		for idx, column := range s.Columns {
			if idx > 0 {
				builder.WriteByte(',')
			}
			builder.WriteQuoted(column)
		}
	} else {
		builder.WriteByte('*')
	}
}

func (s Select) MergeClause(clause *Clause) {
	if v, ok := clause.Expression.(Select); ok {
		s.Columns = append(v.Columns, s.Columns...)
		s.Omits = append(v.Omits, s.Omits...)
	}
	clause.Expression = s
}
