package clause_test

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"gorm.io/gorm/clause"
)

func TestSet(t *testing.T) {
	results := []struct {
		Clauses []clause.Interface
		Result  string
		Vars    []interface{}
	}{
		{
			[]clause.Interface{
				clause.Update{},
				clause.Set([]clause.Assignment{{clause.PrimaryColumn, 1}}),
			},
			"UPDATE `users` SET `users`.`id`=?", []interface{}{1},
		},
		{
			[]clause.Interface{
				clause.Update{},
				clause.Set([]clause.Assignment{{clause.PrimaryColumn, 1}}),
				clause.Set([]clause.Assignment{{clause.Column{Name: "name"}, "jinzhu"}}),
			},
			"UPDATE `users` SET `users`.`id`=?,`name`=?", []interface{}{1, "jinzhu"},
		},
	}

	for idx, result := range results {
		t.Run(fmt.Sprintf("case #%v", idx), func(t *testing.T) {
			checkBuildClauses(t, result.Clauses, result.Result, result.Vars)
		})
	}
}

func TestAssignments(t *testing.T) {
	set := clause.Assignments(map[string]interface{}{
		"name": "jinzhu",
		"age":  18,
	})

	assignments := []clause.Assignment(set)

	sort.Slice(assignments, func(i, j int) bool {
		return strings.Compare(assignments[i].Column.Name, assignments[j].Column.Name) > 0
	})

	if len(assignments) != 2 || assignments[0].Column.Name != "name" || assignments[0].Value.(string) != "jinzhu" || assignments[1].Column.Name != "age" || assignments[1].Value.(int) != 18 {
		t.Errorf("invalid assignments, got %v", assignments)
	}
}

func TestMergeClause(t *testing.T) {
	set1 := clause.Assignments(map[string]interface{}{
		"name": "jinzhu",
	})
	set2 := clause.Assignments(map[string]interface{}{
		"age": 18,
	})

	c := &clause.Clause{
		Name:       "Set",
		Expression: set1,
	}
	set2.MergeClause(c)

	if set3, ok := c.Expression.(clause.Set); ok {
		if len(set3) == 2 {
			return
		}
	}
	t.Error("invalid merge set")
}
