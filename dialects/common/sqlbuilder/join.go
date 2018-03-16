package sqlbuilder

import (
	"github.com/jinzhu/gorm"
)

// BuildJoinCondition build join condition
func BuildJoinCondition(tx *gorm.DB) chan *Builder {
	joinChan := make(chan *Builder)

	go func() {
		builder := &Builder{}
		for _, join := range tx.Statement.Joins {
			if join.Table == "" {
				for _, cond := range join.Conditions {
					buildCondition(tx, cond, builder)
				}
			}
			// FIXME fix join builder
		}
		joinChan <- builder
	}()

	return joinChan
}
