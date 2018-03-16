package sqlbuilder

import (
	"fmt"

	"github.com/jinzhu/gorm"
)

// ConditionInterface condition interface
type ConditionInterface interface {
	ToSQL(*gorm.DB) (string, []interface{})
}

// BuildConditions build conditions
func BuildConditions(tx *gorm.DB) chan *Builder {
	queryChan := make(chan *Builder)

	go func() {
		builder := &Builder{}
		if len(tx.Statement.Conditions) > 0 {
			builder.SQL.WriteString(" WHERE ")
			for i, c := range tx.Statement.Conditions {
				if i > 0 {
					builder.SQL.WriteString(" AND ")
				}
				buildCondition(tx, c, builder)
			}
		}

		queryChan <- builder
	}()
	return queryChan
}

func buildCondition(tx *gorm.DB, c gorm.ConditionInterface, builder *Builder) {
	switch cond := c.(type) {
	case gorm.And:
		builder.SQL.WriteString("(")
		for i, v := range cond {
			if i > 0 {
				builder.SQL.WriteString(" AND ")
			}
			buildCondition(tx, v, builder)
		}
		builder.SQL.WriteString(")")
	case gorm.Or:
		builder.SQL.WriteString("(")
		for i, v := range cond {
			if i > 0 {
				builder.SQL.WriteString(" OR ")
			}
			buildCondition(tx, v, builder)
		}
		builder.SQL.WriteString(")")
	case gorm.Not:
		builder.SQL.WriteString("NOT (")
		for i, v := range cond {
			if i > 0 {
				builder.SQL.WriteString(" AND ")
			}
			buildCondition(tx, v, builder)
		}
		builder.SQL.WriteString(")")
	case gorm.Raw:
		builder.SQL.WriteString(cond.SQL)
		builder.Args = append(builder.Args, cond.Args...)
	case gorm.Eq:
		if cond.Value == nil {
			builder.SQL.WriteString(tx.Dialect().Quote(cond.Column))
			builder.SQL.WriteString(" IS NULL")
		} else {
			builder.SQL.WriteString(tx.Dialect().Quote(cond.Column))
			builder.SQL.WriteString(" = ?")
			builder.Args = append(builder.Args, cond.Value)
		}
	case gorm.Neq:
		if cond.Value == nil {
			builder.SQL.WriteString(tx.Dialect().Quote(cond.Column))
			builder.SQL.WriteString(" IS NOT NULL")
		} else {
			builder.SQL.WriteString(tx.Dialect().Quote(cond.Column))
			builder.SQL.WriteString(" <> ?")
			builder.Args = append(builder.Args, cond.Value)
		}
	case gorm.Gt:
		builder.SQL.WriteString(tx.Dialect().Quote(cond.Column))
		builder.SQL.WriteString(" > ?")
		builder.Args = append(builder.Args, cond.Value)
	case gorm.Gte:
		builder.SQL.WriteString(tx.Dialect().Quote(cond.Column))
		builder.SQL.WriteString(" >= ?")
		builder.Args = append(builder.Args, cond.Value)
	case gorm.Lt:
		builder.SQL.WriteString(tx.Dialect().Quote(cond.Column))
		builder.SQL.WriteString(" < ?")
		builder.Args = append(builder.Args, cond.Value)
	case gorm.Lte:
		builder.SQL.WriteString(tx.Dialect().Quote(cond.Column))
		builder.SQL.WriteString(" <= ?")
		builder.Args = append(builder.Args, cond.Value)
	default:
		if sqlCond, ok := cond.(ConditionInterface); ok {
			sql, as := sqlCond.ToSQL(tx)
			builder.SQL.WriteString(sql)
			builder.Args = append(builder.Args, as...)
		} else {
			tx.AddError(fmt.Errorf("unsupported condition: %#v", cond))
		}
	}

	return
}
