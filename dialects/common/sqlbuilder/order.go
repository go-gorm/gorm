package sqlbuilder

import "github.com/jinzhu/gorm"

// BuildOrderCondition build order condition
func BuildOrderCondition(tx *gorm.DB) chan *Builder {
	orderChan := make(chan *Builder)

	go func() {
		builder := &Builder{}

		if orderBy := tx.Statement.OrderBy; len(orderBy) > 0 {
			builder.SQL.WriteString(" ORDER BY ")
			for i, by := range orderBy {
				if i > 0 {
					builder.SQL.WriteString(", ")
				}
				if str, ok := by.(string); ok {
					builder.SQL.WriteString(str)
				} else {
					buildCondition(tx, by, builder)
				}
			}
		}

		orderChan <- builder
	}()

	return orderChan
}
