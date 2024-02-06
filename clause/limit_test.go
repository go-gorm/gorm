package clause_test

import (
	"fmt"
	"testing"

	"gorm.io/gorm/clause"
)

func TestLimit(t *testing.T) {
	limit0 := 0
	limit10 := 10
	limit50 := 50
	limitNeg10 := -10
	results := []struct {
		Clauses []clause.Interface
		Result  string
		Vars    []interface{}
	}{
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Limit{
				Limit:  &limit10,
				Offset: 20,
			}},
			"SELECT * FROM `users` LIMIT ? OFFSET ?",
			[]interface{}{limit10, 20},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Limit{Limit: &limit0}},
			"SELECT * FROM `users` LIMIT ?",
			[]interface{}{limit0},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Limit{Limit: &limit0}, clause.Limit{Offset: 0}},
			"SELECT * FROM `users` LIMIT ?",
			[]interface{}{limit0},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Limit{Offset: 20}},
			"SELECT * FROM `users` OFFSET ?",
			[]interface{}{20},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Limit{Offset: 20}, clause.Limit{Offset: 30}},
			"SELECT * FROM `users` OFFSET ?",
			[]interface{}{30},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Limit{Offset: 20}, clause.Limit{Limit: &limit10}},
			"SELECT * FROM `users` LIMIT ? OFFSET ?",
			[]interface{}{limit10, 20},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Limit{Limit: &limit10, Offset: 20}, clause.Limit{Offset: 30}},
			"SELECT * FROM `users` LIMIT ? OFFSET ?",
			[]interface{}{limit10, 30},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Limit{Limit: &limit10, Offset: 20}, clause.Limit{Offset: 30}, clause.Limit{Offset: -10}},
			"SELECT * FROM `users` LIMIT ?",
			[]interface{}{limit10},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Limit{Limit: &limit10, Offset: 20}, clause.Limit{Offset: 30}, clause.Limit{Limit: &limitNeg10}},
			"SELECT * FROM `users` OFFSET ?",
			[]interface{}{30},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Limit{Limit: &limit10, Offset: 20}, clause.Limit{Offset: 30}, clause.Limit{Limit: &limit50}},
			"SELECT * FROM `users` LIMIT ? OFFSET ?",
			[]interface{}{limit50, 30},
		},
	}

	for idx, result := range results {
		t.Run(fmt.Sprintf("case #%v", idx), func(t *testing.T) {
			checkBuildClauses(t, result.Clauses, result.Result, result.Vars)
		})
	}
}
