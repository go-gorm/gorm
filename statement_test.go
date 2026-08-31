package gorm

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"gorm.io/gorm/clause"
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

func TestNilCondition(t *testing.T) {
	s := new(Statement)
	if len(s.BuildCondition(nil)) != 0 {
		t.Errorf("Nil condition should be empty")
	}
}

func TestBuildConditionValues(t *testing.T) {
	type namedBytes []byte

	for _, tc := range []struct {
		name      string
		cond      interface{}
		wantCount int
	}{
		{"byte slice is one value", []byte{1, 2, 3}, 1},
		{"named byte slice is one value", namedBytes{1, 2, 3}, 1},
		{"uint slice is a list", []uint{1, 2, 3}, 3},
		{"uint array is a list", [3]uint{1, 2, 3}, 3},
		{"string slice is a list", []string{"a", "b"}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &Statement{DB: &DB{Config: &Config{
				NamingStrategy: schema.NamingStrategy{},
				cacheStore:     &sync.Map{},
			}}}

			exprs := s.BuildCondition(tc.cond)
			if len(exprs) != 1 {
				t.Fatalf("expected 1 expression, got %d", len(exprs))
			}
			in, ok := exprs[0].(clause.IN)
			if !ok {
				t.Fatalf("expected clause.IN, got %T", exprs[0])
			}
			if len(in.Values) != tc.wantCount {
				t.Errorf("expected %d value(s), got %d: %v", tc.wantCount, len(in.Values), in.Values)
			}
		})
	}
}

func TestNameMatcher(t *testing.T) {
	for k, v := range map[string][]string{
		"table.name":         {"table", "name"},
		"`table`.`name`":     {"table", "name"},
		"'table'.'name'":     {"table", "name"},
		"'table'.name":       {"table", "name"},
		"table1.name_23":     {"table1", "name_23"},
		"`table_1`.`name23`": {"table_1", "name23"},
		"'table23'.'name_1'": {"table23", "name_1"},
		"'table23'.name1":    {"table23", "name1"},
		"'name1'":            {"", "name1"},
		"`name_1`":           {"", "name_1"},
		"`Name_1`":           {"", "Name_1"},
		"`Table`.`nAme`":     {"Table", "nAme"},
		"my_table.*":         {"my_table", "*"},
		"`my_table`.*":       {"my_table", "*"},
		"User__Company.*":    {"User__Company", "*"},
		"`User__Company`.*":  {"User__Company", "*"},
		`"User__Company".*`:  {"User__Company", "*"},
		`"table"."*"`:        {"", ""},
	} {
		if table, column := matchName(k); table != v[0] || column != v[1] {
			t.Errorf("failed to match value: %v, got %v, expect: %v", k, []string{table, column}, v)
		}
	}
}
