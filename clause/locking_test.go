package clause_test

import (
	"fmt"
	"testing"

	"github.com/jinzhu/gorm/clause"
)

func TestFor(t *testing.T) {
	results := []struct {
		Clauses []clause.Interface
		Result  string
		Vars    []interface{}
	}{
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.For{
				Lockings: []clause.Locking{{Strength: "UPDATE"}},
			}},
			"SELECT * FROM `users` FOR UPDATE", nil,
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.For{
				Lockings: []clause.Locking{{Strength: "UPDATE"}, {Strength: "SHARE", Table: clause.Table{Name: clause.CurrentTable}}},
			}},
			"SELECT * FROM `users` FOR UPDATE FOR SHARE OF `users`", nil,
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.For{
				Lockings: []clause.Locking{{Strength: "UPDATE"}, {Strength: "SHARE", Table: clause.Table{Name: clause.CurrentTable}}},
			}, clause.For{
				Lockings: []clause.Locking{{Strength: "UPDATE", Options: "NOWAIT"}},
			}},
			"SELECT * FROM `users` FOR UPDATE FOR SHARE OF `users` FOR UPDATE NOWAIT", nil,
		},
	}

	for idx, result := range results {
		t.Run(fmt.Sprintf("case #%v", idx), func(t *testing.T) {
			checkBuildClauses(t, result.Clauses, result.Result, result.Vars)
		})
	}
}
