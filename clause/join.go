package clause

// Join join clause
type Join struct {
	Table From   // From
	Type  string // INNER, LEFT, RIGHT, FULL, CROSS JOIN
	Using []Column
	ON    Where
}

// TODO multiple joins

func (join Join) Build(builder Builder) {
	// TODO
}

func (join Join) MergeExpression(expr Expression) {
	// if j, ok := expr.(Join); ok {
	// 	join.builders = append(join.builders, j.builders...)
	// } else {
	// 	join.builders = append(join.builders, expr)
	// }
}
