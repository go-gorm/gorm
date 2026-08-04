package clause_test

import (
	"fmt"
	"testing"

	"gorm.io/gorm/clause"
)

func TestWhere(t *testing.T) {
	results := []struct {
		Clauses []clause.Interface
		Result  string
		Vars    []interface{}
	}{
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{clause.Eq{Column: clause.PrimaryColumn, Value: "1"}, clause.Gt{Column: "age", Value: 18}, clause.Or(clause.Neq{Column: "name", Value: "jinzhu"})},
			}},
			"SELECT * FROM `users` WHERE `users`.`id` = ? AND `age` > ? OR `name` <> ?",
			[]interface{}{"1", 18, "jinzhu"},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{clause.Or(clause.Neq{Column: "name", Value: "jinzhu"}), clause.Eq{Column: clause.PrimaryColumn, Value: "1"}, clause.Gt{Column: "age", Value: 18}},
			}},
			"SELECT * FROM `users` WHERE `users`.`id` = ? OR `name` <> ? AND `age` > ?",
			[]interface{}{"1", "jinzhu", 18},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{clause.Or(clause.Neq{Column: "name", Value: "jinzhu"}), clause.Eq{Column: clause.PrimaryColumn, Value: "1"}, clause.Gt{Column: "age", Value: 18}},
			}},
			"SELECT * FROM `users` WHERE `users`.`id` = ? OR `name` <> ? AND `age` > ?",
			[]interface{}{"1", "jinzhu", 18},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{clause.Or(clause.Eq{Column: clause.PrimaryColumn, Value: "1"}), clause.Or(clause.Neq{Column: "name", Value: "jinzhu"})},
			}},
			"SELECT * FROM `users` WHERE `users`.`id` = ? OR `name` <> ?",
			[]interface{}{"1", "jinzhu"},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{clause.Eq{Column: clause.PrimaryColumn, Value: "1"}, clause.Gt{Column: "age", Value: 18}, clause.Or(clause.Neq{Column: "name", Value: "jinzhu"})},
			}, clause.Where{
				Exprs: []clause.Expression{clause.Or(clause.Gt{Column: "score", Value: 100}, clause.Like{Column: "name", Value: "%linus%"})},
			}},
			"SELECT * FROM `users` WHERE `users`.`id` = ? AND `age` > ? OR `name` <> ? AND (`score` > ? OR `name` LIKE ?)",
			[]interface{}{"1", 18, "jinzhu", 100, "%linus%"},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{clause.Not(clause.Eq{Column: clause.PrimaryColumn, Value: "1"}, clause.Gt{Column: "age", Value: 18}), clause.Or(clause.Neq{Column: "name", Value: "jinzhu"})},
			}, clause.Where{
				Exprs: []clause.Expression{clause.Or(clause.Not(clause.Gt{Column: "score", Value: 100}), clause.Like{Column: "name", Value: "%linus%"})},
			}},
			"SELECT * FROM `users` WHERE (`users`.`id` <> ? AND `age` <= ?) OR `name` <> ? AND (`score` <= ? OR `name` LIKE ?)",
			[]interface{}{"1", 18, "jinzhu", 100, "%linus%"},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{clause.And(clause.Eq{Column: "age", Value: 18}, clause.Or(clause.Neq{Column: "name", Value: "jinzhu"}))},
			}},
			"SELECT * FROM `users` WHERE `age` = ? OR `name` <> ?",
			[]interface{}{18, "jinzhu"},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{clause.Not(clause.Eq{Column: clause.PrimaryColumn, Value: "1"}, clause.Gt{Column: "age", Value: 18}), clause.And(clause.Expr{SQL: "`score` <= ?", Vars: []interface{}{100}, WithoutParentheses: false})},
			}},
			"SELECT * FROM `users` WHERE (`users`.`id` <> ? AND `age` <= ?) AND `score` <= ?",
			[]interface{}{"1", 18, 100},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{clause.Not(clause.Eq{Column: clause.PrimaryColumn, Value: "1"}, clause.Gt{Column: "age", Value: 18}), clause.Expr{SQL: "`score` <= ?", Vars: []interface{}{100}, WithoutParentheses: false}},
			}},
			"SELECT * FROM `users` WHERE (`users`.`id` <> ? AND `age` <= ?) AND `score` <= ?",
			[]interface{}{"1", 18, 100},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{clause.Not(clause.Eq{Column: clause.PrimaryColumn, Value: "1"}, clause.Gt{Column: "age", Value: 18}), clause.Or(clause.Expr{SQL: "`score` <= ?", Vars: []interface{}{100}, WithoutParentheses: false})},
			}},
			"SELECT * FROM `users` WHERE (`users`.`id` <> ? AND `age` <= ?) OR `score` <= ?",
			[]interface{}{"1", 18, 100},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{
					clause.And(clause.Not(clause.Eq{Column: clause.PrimaryColumn, Value: "1"}),
						clause.And(clause.Expr{SQL: "`score` <= ?", Vars: []interface{}{100}, WithoutParentheses: false})),
				},
			}},
			"SELECT * FROM `users` WHERE `users`.`id` <> ? AND `score` <= ?",
			[]interface{}{"1", 100},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{clause.Not(clause.Eq{Column: clause.PrimaryColumn, Value: "1"},
					clause.And(clause.Expr{SQL: "`score` <= ?", Vars: []interface{}{100}, WithoutParentheses: false}))},
			}},
			"SELECT * FROM `users` WHERE (`users`.`id` <> ? AND NOT `score` <= ?)",
			[]interface{}{"1", 100},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{clause.Not(clause.Expr{SQL: "`score` <= ?", Vars: []interface{}{100}},
					clause.Expr{SQL: "`age` <= ?", Vars: []interface{}{60}})},
			}},
			"SELECT * FROM `users` WHERE NOT (`score` <= ? AND `age` <= ?)",
			[]interface{}{100, 60},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{
					clause.Not(clause.AndConditions{
						Exprs: []clause.Expression{
							clause.Eq{Column: clause.PrimaryColumn, Value: "1"},
							clause.Gt{Column: "age", Value: 18},
						}}, clause.OrConditions{
						Exprs: []clause.Expression{
							clause.Lt{Column: "score", Value: 100},
						},
					}),
				}}},
			"SELECT * FROM `users` WHERE NOT ((`users`.`id` = ? AND `age` > ?) OR `score` < ?)",
			[]interface{}{"1", 18, 100},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "birthday IS NULL\n\tOR birthday < CURRENT_TIMESTAMP"},
					clause.Eq{Column: "name", Value: "jinzhu"},
				},
			}},
			"SELECT * FROM `users` WHERE (birthday IS NULL\n\tOR birthday < CURRENT_TIMESTAMP) AND `name` = ?",
			[]interface{}{"jinzhu"},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "a = 1 OR b = 2"},
					clause.Eq{Column: "name", Value: "jinzhu"},
				},
			}},
			"SELECT * FROM `users` WHERE (a = 1 OR b = 2) AND `name` = ?",
			[]interface{}{"jinzhu"},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{
					clause.NamedExpr{SQL: "a = 1\n\tOR b = 2"},
					clause.Eq{Column: "name", Value: "jinzhu"},
				},
			}},
			"SELECT * FROM `users` WHERE (a = 1\n\tOR b = 2) AND `name` = ?",
			[]interface{}{"jinzhu"},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{clause.Not(clause.Expr{SQL: "a = 1\n\tOR b = 2"})},
			}},
			"SELECT * FROM `users` WHERE NOT (a = 1\n\tOR b = 2)",
			nil,
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "floor(a) = 1"},
					clause.Eq{Column: "name", Value: "jinzhu"},
				},
			}},
			"SELECT * FROM `users` WHERE floor(a) = 1 AND `name` = ?",
			[]interface{}{"jinzhu"},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "a = 1\tOR b = 2"},
					clause.Eq{Column: "name", Value: "jinzhu"},
				},
			}},
			"SELECT * FROM `users` WHERE (a = 1\tOR b = 2) AND `name` = ?",
			[]interface{}{"jinzhu"},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "a = 1\nAND b = 2"},
					clause.Eq{Column: "name", Value: "jinzhu"},
				},
			}},
			"SELECT * FROM `users` WHERE (a = 1\nAND b = 2) AND `name` = ?",
			[]interface{}{"jinzhu"},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{
					clause.Or(clause.Expr{SQL: "a = 1\n\tOR b = 2"}),
					clause.Eq{Column: "name", Value: "jinzhu"},
				},
			}},
			"SELECT * FROM `users` WHERE `name` = ? OR (a = 1\n\tOR b = 2)",
			[]interface{}{"jinzhu"},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{
					clause.AndConditions{Exprs: []clause.Expression{clause.Expr{SQL: "a = 1\n\tOR b = 2"}}},
					clause.Eq{Column: "name", Value: "jinzhu"},
				},
			}},
			"SELECT * FROM `users` WHERE (a = 1\n\tOR b = 2) AND `name` = ?",
			[]interface{}{"jinzhu"},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{
					clause.Or(clause.Expr{SQL: "a = 1\n\tOR b = 2"}, clause.Expr{SQL: "c = 3\n\tOR d = 4"}),
				},
			}},
			"SELECT * FROM `users` WHERE ((a = 1\n\tOR b = 2) OR (c = 3\n\tOR d = 4))",
			nil,
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{
					clause.Not(clause.Eq{Column: "id", Value: 1}, clause.Expr{SQL: "a = 1\n\tOR b = 2"}),
				},
			}},
			"SELECT * FROM `users` WHERE (`id` <> ? AND NOT (a = 1\n\tOR b = 2))",
			[]interface{}{1},
		},
		{
			[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{
				Exprs: []clause.Expression{
					clause.Expr{SQL: "brand = 'x'"},
					clause.Eq{Column: "name", Value: "jinzhu"},
				},
			}},
			"SELECT * FROM `users` WHERE brand = 'x' AND `name` = ?",
			[]interface{}{"jinzhu"},
		},
	}

	for idx, result := range results {
		t.Run(fmt.Sprintf("case #%v", idx), func(t *testing.T) {
			checkBuildClauses(t, result.Clauses, result.Result, result.Vars)
		})
	}
}
