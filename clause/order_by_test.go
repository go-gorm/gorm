package clause_test

import (
	"fmt"
	"testing"

	"gorm.io/gorm/clause"
)

func TestOrderBy(t *testing.T) {
	results := []struct {
		Clauses []clause.Interface
		Result  string
		Vars    []interface{}
	}{
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.OrderBy{
				Columns: []clause.OrderByColumn{{Column: clause.PrimaryColumn, Desc: true}},
			}},
			"SELECT * FROM `users` ORDER BY `users`.`id` DESC", nil,
		},
		{
			[]clause.Interface{
				clause.Select{}, clause.From{}, clause.OrderBy{
					Columns: []clause.OrderByColumn{{Column: clause.PrimaryColumn, Desc: true}},
				}, clause.OrderBy{
					Columns: []clause.OrderByColumn{{Column: clause.Column{Name: "name"}}},
				},
			},
			"SELECT * FROM `users` ORDER BY `users`.`id` DESC,`name`", nil,
		},
		{
			[]clause.Interface{
				clause.Select{}, clause.From{}, clause.OrderBy{
					Columns: []clause.OrderByColumn{{Column: clause.PrimaryColumn, Desc: true}},
				}, clause.OrderBy{
					Columns: []clause.OrderByColumn{{Column: clause.Column{Name: "name"}, Reorder: true}},
				},
			},
			"SELECT * FROM `users` ORDER BY `name`", nil,
		},
		{
			[]clause.Interface{
				clause.Select{}, clause.From{}, clause.OrderBy{
					Expression: clause.Expr{SQL: "FIELD(id, ?)", Vars: []interface{}{[]int{1, 2, 3}}, WithoutParentheses: true},
				},
			},
			"SELECT * FROM `users` ORDER BY FIELD(id, ?,?,?)",
			[]interface{}{1, 2, 3},
		},
	}

	expression := clause.OrderBy{Expression: clause.Expr{SQL: "name = ?", Vars: []interface{}{"preferred"}}}
	column := clause.OrderBy{Columns: []clause.OrderByColumn{{Column: clause.PrimaryColumn}}}
	reorder := clause.OrderBy{Columns: []clause.OrderByColumn{{Column: clause.PrimaryColumn, Reorder: true}}}
	for _, tc := range []struct {
		name   string
		orders []clause.Interface
		want   string
		vars   []interface{}
	}{
		{"expression then column", []clause.Interface{expression, column}, "name = ?, `users`.`id`", []interface{}{"preferred"}},
		{"column then expression", []clause.Interface{column, expression}, "`users`.`id`, name = ?", []interface{}{"preferred"}},
		{"reorder replaces expression", []clause.Interface{expression, reorder}, "`users`.`id`", nil},
		{"empty orders", []clause.Interface{clause.OrderBy{}, expression, clause.OrderBy{}}, "name = ?", []interface{}{"preferred"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clauses := append([]clause.Interface{clause.Select{}, clause.From{}}, tc.orders...)
			checkBuildClauses(t, clauses, "SELECT * FROM `users` ORDER BY "+tc.want, tc.vars)
		})
	}

	for idx, result := range results {
		t.Run(fmt.Sprintf("case #%v", idx), func(t *testing.T) {
			checkBuildClauses(t, result.Clauses, result.Result, result.Vars)
		})
	}
}
