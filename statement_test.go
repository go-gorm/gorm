package gorm

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func TestWhereCloneCorruption(t *testing.T) {
	for whereCount := 1; whereCount <= 8; whereCount++ {
		t.Run(fmt.Sprintf("w=%d", whereCount), func(t *testing.T) {
			s := new(Statement)
			for w := 0; w < whereCount; w++ {
				s = s.clone()
				s.AddClause(clause.Where{
					Exprs: s.BuildCondition(fmt.Sprintf("where%d", w)),
				})
			}

			s1 := s.clone()
			s1.AddClause(clause.Where{
				Exprs: s.BuildCondition("FINAL1"),
			})
			s2 := s.clone()
			s2.AddClause(clause.Where{
				Exprs: s.BuildCondition("FINAL2"),
			})

			if reflect.DeepEqual(s1.Clauses["WHERE"], s2.Clauses["WHERE"]) {
				t.Errorf("Where conditions should be different")
			}
		})
	}
}

var _ Dialector = new(dummyDialector)

type dummyDialector struct{}

func (dummyDialector) Name() string         { return "dummy" }
func (dummyDialector) Initialize(*DB) error { return nil }
func (dummyDialector) DefaultValueOf(field *schema.Field) clause.Expression {
	return clause.Expr{SQL: "DEFAULT"}
}
func (dummyDialector) Migrator(*DB) Migrator { return nil }
func (dummyDialector) BindVarTo(writer clause.Writer, stmt *Statement, v interface{}) {
	writer.WriteByte('?')
}
func (dummyDialector) QuoteTo(writer clause.Writer, str string) { writer.WriteString("`" + str + "`") }
func (dummyDialector) Explain(sql string, vars ...interface{}) string {
	return logger.ExplainSQL(sql, nil, `"`, vars...)
}
func (dummyDialector) DataTypeOf(*schema.Field) string { return "" }

var db, _ = Open(dummyDialector{})

func TestStatement_WriteQuoted(t *testing.T) {
	s := Statement{DB: db}

	testdata := map[string]clause.Expression{
		"SUM(`users`.`id`)": clause.Expr{SQL: "SUM(?)", Vars: []interface{}{clause.Column{Table: "users", Name: "id"}}},
	}

	for result, expr := range testdata {
		s.WriteQuoted(expr)
		if s.SQL.String() != result {
			t.Errorf("WriteQuoted test fail, expected: %s, got %s", result, s.SQL.String())
		}

		s.SQL = strings.Builder{}
		s.Vars = []interface{}{}
	}
}
