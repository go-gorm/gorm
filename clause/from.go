package clause

// From from clause
type From struct {
	Tables []Table
	Joins  []Join
}

type JoinType string

const (
	CrossJoin JoinType = "CROSS"
	InnerJoin          = "INNER"
	LeftJoin           = "LEFT"
	RightJoin          = "RIGHT"
)

// Join join clause for from
type Join struct {
	Type  JoinType
	Table Table
	ON    Where
	Using []string
}

// Name from clause name
func (from From) Name() string {
	return "FROM"
}

// Build build from clause
func (from From) Build(builder Builder) {
	if len(from.Tables) > 0 {
		for idx, table := range from.Tables {
			if idx > 0 {
				builder.WriteByte(',')
			}

			builder.WriteQuoted(table)
		}
	} else {
		builder.WriteQuoted(currentTable)
	}

	for _, join := range from.Joins {
		builder.WriteByte(' ')
		join.Build(builder)
	}
}

func (join Join) Build(builder Builder) {
	if join.Type != "" {
		builder.Write(string(join.Type))
		builder.WriteByte(' ')
	}

	builder.Write("JOIN ")
	builder.WriteQuoted(join.Table)

	if len(join.ON.Exprs) > 0 {
		builder.Write(" ON ")
		join.ON.Build(builder)
	} else if len(join.Using) > 0 {
		builder.Write(" USING (")
		for idx, c := range join.Using {
			if idx > 0 {
				builder.WriteByte(',')
			}
			builder.WriteQuoted(c)
		}
		builder.WriteByte(')')
	}
}

// MergeClause merge from clause
func (from From) MergeClause(clause *Clause) {
	if v, ok := clause.Expression.(From); ok {
		from.Tables = append(v.Tables, from.Tables...)
		from.Joins = append(v.Joins, from.Joins...)
	}
	clause.Expression = from
}
