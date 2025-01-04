package clause_test

import (
	"fmt"
	"math"
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
		// case #0 - limit10 offset20
		{
			[]clause.Interface{
				clause.Select{}, clause.From{},
				clause.Limit{Limit: &limit10, Offset: 20},
			},
			"SELECT * FROM `users` LIMIT ? OFFSET ?",
			[]interface{}{limit10, 20},
		},
		// case #1 - limit0
		{
			[]clause.Interface{
				clause.Select{}, clause.From{},
				clause.Limit{Limit: &limit0},
			},
			"SELECT * FROM `users` LIMIT ?",
			[]interface{}{limit0},
		},
		// case #2 - limit0 offset0 => offset0 is effectively ignored
		{
			[]clause.Interface{
				clause.Select{}, clause.From{},
				clause.Limit{Limit: &limit0}, clause.Limit{Offset: 0},
			},
			"SELECT * FROM `users` LIMIT ?",
			[]interface{}{limit0},
		},
		// case #3 - only offset=20
		// MySQL demands limit if offset>0 => math.MaxInt
		{
			[]clause.Interface{
				clause.Select{}, clause.From{},
				clause.Limit{Offset: 20},
			},
			"SELECT * FROM `users` LIMIT ? OFFSET ?",
			[]interface{}{math.MaxInt, 20},
		},
		// case #4 - multiple offsets (20 -> 30)
		{
			[]clause.Interface{
				clause.Select{}, clause.From{},
				clause.Limit{Offset: 20}, clause.Limit{Offset: 30},
			},
			"SELECT * FROM `users` LIMIT ? OFFSET ?",
			[]interface{}{math.MaxInt, 30},
		},
		// case #5 - merge offset20 with limit10
		{
			[]clause.Interface{
				clause.Select{}, clause.From{},
				clause.Limit{Offset: 20}, clause.Limit{Limit: &limit10},
			},
			"SELECT * FROM `users` LIMIT ? OFFSET ?",
			[]interface{}{limit10, 20},
		},
		// case #6 - merge offset20->30 with limit10
		{
			[]clause.Interface{
				clause.Select{}, clause.From{},
				clause.Limit{Limit: &limit10, Offset: 20},
				clause.Limit{Offset: 30},
			},
			"SELECT * FROM `users` LIMIT ? OFFSET ?",
			[]interface{}{limit10, 30},
		},
		// case #7 - negative offset => offset=0 => "SELECT * FROM `users` LIMIT 10"
		{
			[]clause.Interface{
				clause.Select{}, clause.From{},
				clause.Limit{Limit: &limit10, Offset: 20},
				clause.Limit{Offset: 30},
				clause.Limit{Offset: -10},
			},
			"SELECT * FROM `users` LIMIT ?",
			[]interface{}{limit10},
		},
		// case #8 - negative limit => treat as nil => offset=30 => => "LIMIT ? OFFSET ?"
		{
			[]clause.Interface{
				clause.Select{}, clause.From{},
				// Start with limit10 offset20
				clause.Limit{Limit: &limit10, Offset: 20},
				// Then change offset to 30
				clause.Limit{Offset: 30},
				// Then set limit to negative => remove limit => offset>0 => limit=math.MaxInt
				clause.Limit{Limit: &limitNeg10},
			},
			"SELECT * FROM `users` LIMIT ? OFFSET ?",
			[]interface{}{math.MaxInt, 30},
		},
		// case #9 - changing limit from 10 -> 50, offset=30
		{
			[]clause.Interface{
				clause.Select{}, clause.From{},
				clause.Limit{Limit: &limit10, Offset: 20},
				clause.Limit{Offset: 30},
				clause.Limit{Limit: &limit50},
			},
			"SELECT * FROM `users` LIMIT ? OFFSET ?",
			[]interface{}{limit50, 30},
		},
		// case #10 - only offset=100 => "LIMIT ? OFFSET ?", math.MaxInt, 100
		{
			[]clause.Interface{
				clause.Select{}, clause.From{},
				clause.Limit{Offset: 100},
			},
			"SELECT * FROM `users` LIMIT ? OFFSET ?",
			[]interface{}{math.MaxInt, 100},
		},
	}

	for idx, result := range results {
		t.Run(fmt.Sprintf("case #%v", idx), func(t *testing.T) {
			checkBuildClauses(t, result.Clauses, result.Result, result.Vars)
		})
	}
}
