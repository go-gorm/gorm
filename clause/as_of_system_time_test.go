package clause_test

import (
	"fmt"
	"testing"
	"time"

	"gorm.io/gorm/clause"
)

func TestAsOfSystemTime(t *testing.T) {
	results := []struct {
		Clauses []clause.Interface
		Result  string
		Vars    []interface{}
	}{
		{
			[]clause.Interface{
				clause.Select{},
				clause.From{
					Tables:         []clause.Table{{Name: "users"}},
					AsOfSystemTime: &clause.AsOfSystemTime{Timestamp: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)},
				},
			},
			"SELECT * FROM `users` AS OF SYSTEM TIME '2023-01-01 12:00:00.000000'",
			nil,
		},
		{
			[]clause.Interface{
				clause.Select{},
				clause.From{
					Tables:         []clause.Table{{Name: "users"}},
					AsOfSystemTime: &clause.AsOfSystemTime{Raw: "-1h"},
				},
			},
			"SELECT * FROM `users` AS OF SYSTEM TIME '-1h'",
			nil,
		},
		{
			[]clause.Interface{
				clause.Select{},
				clause.From{
					Tables: []clause.Table{{Name: "users"}},
					Joins: []clause.Join{
						{
							Type:  clause.InnerJoin,
							Table: clause.Table{Name: "companies"},
							ON: clause.Where{
								[]clause.Expression{clause.Eq{clause.Column{Table: "companies", Name: "id"}, clause.Column{Table: "users", Name: "company_id"}}},
							},
						},
					},
					AsOfSystemTime: &clause.AsOfSystemTime{Raw: "-1h"},
				},
			},
			"SELECT * FROM `users` INNER JOIN `companies` ON `companies`.`id` = `users`.`company_id` AS OF SYSTEM TIME '-1h'",
			nil,
		},
	}

	for idx, result := range results {
		t.Run(fmt.Sprintf("case #%v", idx), func(t *testing.T) {
			checkBuildClauses(t, result.Clauses, result.Result, result.Vars)
		})
	}
}

func TestAsOfSystemTimeName(t *testing.T) {
	asOfClause := clause.AsOfSystemTime{}
	if asOfClause.Name() != "AS OF SYSTEM TIME" {
		t.Errorf("expected name 'AS OF SYSTEM TIME', got %q", asOfClause.Name())
	}
}

func TestAsOfSystemTimeMergeClause(t *testing.T) {
	asOfClause := clause.AsOfSystemTime{Timestamp: time.Now()}
	var c clause.Clause
	asOfClause.MergeClause(&c)

	if c.Expression != asOfClause {
		t.Error("expected expression to be set to the clause")
	}
}
