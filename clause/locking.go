package clause

type For struct {
	Lockings []Locking
}

type Locking struct {
	Strength string
	Table    Table
	Options  string
}

// Name where clause name
func (f For) Name() string {
	return "FOR"
}

// Build build where clause
func (f For) Build(builder Builder) {
	for idx, locking := range f.Lockings {
		if idx > 0 {
			builder.WriteByte(' ')
		}

		builder.WriteString("FOR ")
		builder.WriteString(locking.Strength)
		if locking.Table.Name != "" {
			builder.WriteString(" OF ")
			builder.WriteQuoted(locking.Table)
		}

		if locking.Options != "" {
			builder.WriteByte(' ')
			builder.WriteString(locking.Options)
		}
	}
}

// MergeClause merge order by clauses
func (f For) MergeClause(clause *Clause) {
	clause.Name = ""

	if v, ok := clause.Expression.(For); ok {
		f.Lockings = append(v.Lockings, f.Lockings...)
	}

	clause.Expression = f
}
