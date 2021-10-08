package gorm

import (
	"fmt"
	"reflect"
	"testing"

	"gorm.io/gorm/clause"
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

func TestNameMatcher(t *testing.T) {
	for k, v := range map[string]string{
		"table.name":     "name",
		"`table`.`name`": "name",
		"'table'.'name'": "name",
		"'table'.name":   "name",
	} {
		if matches := nameMatcher.FindStringSubmatch(k); len(matches) < 2 || matches[1] != v {
			t.Errorf("failed to match value: %v, got %v, expect: %v", k, matches, v)
		}
	}
}
