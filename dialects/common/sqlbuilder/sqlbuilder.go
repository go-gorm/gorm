package sqlbuilder

import (
	"bytes"
	"fmt"

	"github.com/jinzhu/gorm"
)

func buildCondition(tx *gorm.DB, c gorm.ConditionInterface, s *bytes.Buffer) []interface{} {
	args := []interface{}{}

	switch cond := c.(type) {
	case gorm.And:
		s.WriteString("(")
		for i, v := range cond {
			if i > 0 {
				s.WriteString(" AND ")
			}
			args = append(args, buildCondition(tx, v, s)...)
		}
		s.WriteString(")")
	case gorm.Or:
		s.WriteString("(")
		for i, v := range cond {
			if i > 0 {
				s.WriteString(" OR ")
			}
			args = append(args, buildCondition(tx, v, s)...)
		}
		s.WriteString(")")
	case gorm.Not:
		s.WriteString("NOT (")
		for i, v := range cond {
			if i > 0 {
				s.WriteString(" AND ")
			}
			args = append(args, buildCondition(tx, v, s)...)
		}
		s.WriteString(")")
	case gorm.Raw:
		s.WriteString(cond.SQL)
		args = append(args, cond.Args...)
	case gorm.Eq:
		if cond.Value == nil {
			s.WriteString(tx.Dialect().Quote(cond.Column))
			s.WriteString(" IS NULL")
		} else {
			s.WriteString(tx.Dialect().Quote(cond.Column))
			s.WriteString(" = ?")
			args = append(args, cond.Value)
		}
	case gorm.Neq:
		if cond.Value == nil {
			s.WriteString(tx.Dialect().Quote(cond.Column))
			s.WriteString(" IS NOT NULL")
		} else {
			s.WriteString(tx.Dialect().Quote(cond.Column))
			s.WriteString(" <> ?")
			args = append(args, cond.Value)
		}
	case gorm.Gt:
		s.WriteString(tx.Dialect().Quote(cond.Column))
		s.WriteString(" > ?")
		args = append(args, cond.Value)
	case gorm.Gte:
		s.WriteString(tx.Dialect().Quote(cond.Column))
		s.WriteString(" >= ?")
		args = append(args, cond.Value)
	case gorm.Lt:
		s.WriteString(tx.Dialect().Quote(cond.Column))
		s.WriteString(" < ?")
		args = append(args, cond.Value)
	case gorm.Lte:
		s.WriteString(tx.Dialect().Quote(cond.Column))
		s.WriteString(" <= ?")
		args = append(args, cond.Value)
	default:
		if sqlCond, ok := cond.(ConditionInterface); ok {
			sql, as := sqlCond.ToSQL(tx)
			s.WriteString(sql)
			args = append(args, as)
		} else {
			tx.AddError(fmt.Errorf("unsupported condition: %#v", cond))
		}
	}

	return args
}

// ConditionInterface condition interface
type ConditionInterface interface {
	ToSQL(*gorm.DB) (string, []interface{})
}

// BuildConditions build conditions
func BuildConditions(tx *gorm.DB) chan string {
	queryChan := make(chan string)

	go func() {
		s := bytes.NewBufferString("")
		args := []interface{}{}

		for i, c := range tx.Statement.Conditions {
			if i > 0 {
				s.WriteString(" AND ")
			}
			args = append(args, buildCondition(tx, c, s)...)
		}
	}()
	return queryChan
}
