package sqlbuilder

import (
	"fmt"

	"github.com/jinzhu/gorm"
)

// BuildLimitCondition build limit condition
func BuildLimitCondition(tx *gorm.DB) chan *Builder {
	limitChan := make(chan *Builder)

	go func() {
		builder := &Builder{}
		if limit := tx.Statement.Limit; limit.Limit != nil {
			builder.SQL.WriteString(fmt.Sprintf(" LIMIT %d", *limit.Limit))

			if limit.Offset != nil {
				builder.SQL.WriteString(fmt.Sprintf(" OFFSET %d", *limit.Offset))
			}
		}
		limitChan <- builder
	}()

	return limitChan
}
