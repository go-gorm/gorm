package clause_test

import (
	"fmt"
	"testing"

	"gorm.io/gorm/clause"
)

func TestWith(t *testing.T) {
	results := []struct {
		Clauses []clause.Interface
		Result  string
		Vars    []interface{}
	}{
		{
			[]clause.Interface{clause.With{Exprs: []clause.Expression{
				clause.WithExpression{Name: "cte1", Columns: []string{"foo", "bar"}, Expr: clause.Expr{SQL: "SELECT 1, 2"}},
				clause.WithExpression{Name: "cte2", Expr: clause.Expr{SQL: "SELECT 1"}},
			}}},
			"WITH `cte1` (`foo`,`bar`) AS (SELECT 1, 2),`cte2` AS (SELECT 1)",
			nil,
		},
		{
			[]clause.Interface{
				clause.With{
					Recursive: true,
					Exprs:     []clause.Expression{clause.WithExpression{Name: "cte1", Expr: clause.Expr{SQL: "SELECT 1"}}},
				},
				clause.With{
					Recursive: false,
					Exprs:     []clause.Expression{clause.WithExpression{Name: "cte2", Expr: clause.Expr{SQL: "SELECT 2"}}},
				},
			},
			"WITH RECURSIVE `cte1` AS (SELECT 1),`cte2` AS (SELECT 2)",
			nil,
		},
		{
			[]clause.Interface{
				clause.With{
					Expression: clause.Expr{SQL: "`cte1` AS (SELECT 1)"},
				},
				clause.With{
					Exprs: []clause.Expression{clause.WithExpression{Name: "cte2", Expr: clause.Expr{SQL: "SELECT 2"}}},
				},
			},
			"WITH `cte1` AS (SELECT 1)",
			nil,
		},
		{
			[]clause.Interface{
				clause.With{
					Exprs: []clause.Expression{clause.WithExpression{Name: "cte1", Expr: clause.Expr{SQL: "SELECT 1"}}},
				},
				clause.With{
					Expression: clause.Expr{SQL: "`cte2` AS (SELECT 2)"},
				},
			},
			"WITH `cte2` AS (SELECT 2)",
			nil,
		},
		{
			[]clause.Interface{clause.With{
				Exprs: []clause.Expression{
					clause.WithExpression{Name: "cte1", Columns: []string{"foo", "bar"}, Expr: clause.Expr{SQL: "SELECT 1, 2"}},
				},
				Expression: clause.Expr{SQL: "cte2 AS (SELECT 1)"},
			}},
			"WITH cte2 AS (SELECT 1)",
			nil,
		},
		{
			[]clause.Interface{clause.With{
				Recursive:  false,
				Exprs:      nil,
				Expression: nil,
			}},
			"WITH",
			nil,
		},
		{
			[]clause.Interface{clause.With{Exprs: []clause.Expression{clause.WithExpression{
				Name: "",
				Expr: clause.Expr{SQL: "SELECT 1"},
			}}}},
			"WITH",
			nil,
		},
		{
			[]clause.Interface{clause.With{Exprs: []clause.Expression{clause.WithExpression{
				Name: "cte1",
				Expr: nil,
			}}}},
			"WITH",
			nil,
		},
	}

	for idx, result := range results {
		t.Run(fmt.Sprintf("case #%v", idx), func(t *testing.T) {
			checkBuildClauses(t, result.Clauses, result.Result, result.Vars)
		})
	}
}
