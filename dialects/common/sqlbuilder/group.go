package sqlbuilder

import (
	"strings"

	"github.com/jinzhu/gorm"
)

// BuildGroupCondition build group condition
func BuildGroupCondition(tx *gorm.DB) chan *Builder {
	groupChan := make(chan *Builder)

	go func() {
		builder := &Builder{}
		if groupBy := tx.Statement.GroupBy; len(groupBy.Columns) > 0 {
			builder.SQL.WriteString(" GROUP BY ")
			builder.SQL.WriteString(strings.Join(tx.Statement.GroupBy.Columns, ", "))

			if len(groupBy.Having) > 0 {
				builder.SQL.WriteString(" HAVING ")
				for i, having := range groupBy.Having {
					if i > 0 {
						builder.SQL.WriteString(" AND ")
					}
					buildCondition(tx, having, builder)
				}
			}
		}
		groupChan <- builder
	}()

	return groupChan
}
