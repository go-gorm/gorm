package clause

// From from clause
type From struct {
	Tables []Table
}

// Name from clause name
func (From) Name() string {
	return "FROM"
}

// Build build from clause
func (from From) Build(builder Builder) {
	for idx, table := range from.Tables {
		if idx > 0 {
			builder.WriteByte(',')
		}

		builder.WriteQuoted(table)
	}
}
