package clause_test

import (
	"fmt"
	"testing"

	"gorm.io/gorm/clause"
)

func TestLocking(t *testing.T) {
	results := []struct {
		Clauses []clause.Interface
		Result  string
		Vars    []interface{}
	}{
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Locking{Strength: clause.LockingStrengthUpdate}},
			"SELECT * FROM `users` FOR UPDATE", nil,
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Locking{Strength: clause.LockingStrengthShare, Table: clause.Table{Name: clause.CurrentTable}}},
			"SELECT * FROM `users` FOR SHARE OF `users`", nil,
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Locking{Strength: clause.LockingStrengthUpdate, Options: clause.LockingOptionsNoWait}},
			"SELECT * FROM `users` FOR UPDATE NOWAIT", nil,
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Locking{Strength: clause.LockingStrengthUpdate, Options: clause.LockingOptionsSkipLocked}},
			"SELECT * FROM `users` FOR UPDATE SKIP LOCKED", nil,
		},
	}

	for idx, result := range results {
		t.Run(fmt.Sprintf("case #%v", idx), func(t *testing.T) {
			checkBuildClauses(t, result.Clauses, result.Result, result.Vars)
		})
	}
}
