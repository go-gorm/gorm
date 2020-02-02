package clause

// Select select attrs when querying, updating, creating
type Select struct {
	SelectColumns []Column
	OmitColumns   []Column
}

// SelectInterface select clause interface
type SelectInterface interface {
	Selects() []Column
	Omits() []Column
}

func (s Select) Selects() []Column {
	return s.SelectColumns
}

func (s Select) Omits() []Column {
	return s.OmitColumns
}

func (s Select) Build(builder Builder) {
	if len(s.SelectColumns) > 0 {
		for idx, column := range s.SelectColumns {
			if idx > 0 {
				builder.WriteByte(',')
			}
			builder.WriteQuoted(column)
		}
	} else {
		builder.WriteByte('*')
	}
}

func (s Select) MergeExpression(expr Expression) {
	if v, ok := expr.(SelectInterface); ok {
		if len(s.SelectColumns) == 0 {
			s.SelectColumns = v.Selects()
		}
		if len(s.OmitColumns) == 0 {
			s.OmitColumns = v.Omits()
		}
	}
}
