package clause_test

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/clause"
	"github.com/jinzhu/gorm/schema"
	"github.com/jinzhu/gorm/tests"
)

func TestClauses(t *testing.T) {
	var (
		db, _   = gorm.Open(tests.DummyDialector{}, nil)
		results = []struct {
			Clauses []clause.Interface
			Result  string
			Vars    []interface{}
		}{
			{
				[]clause.Interface{clause.Select{}, clause.From{}, clause.Where{AndConditions: []clause.Expression{clause.Eq{Column: clause.PrimaryColumn, Value: "1"}}}},
				"SELECT * FROM `users` WHERE `users`.`id` = ?", []interface{}{"1"},
			},
		}
	)

	for idx, result := range results {
		t.Run(fmt.Sprintf("case #%v", idx), func(t *testing.T) {
			var (
				user, _ = schema.Parse(&tests.User{}, &sync.Map{}, db.NamingStrategy)
				stmt    = gorm.Statement{
					DB: db, Table: user.Table, Schema: user, Clauses: map[string]clause.Clause{},
				}
				buildNames []string
			)

			for _, c := range result.Clauses {
				buildNames = append(buildNames, c.Name())
				stmt.AddClause(c)
			}

			stmt.Build(buildNames...)

			if stmt.SQL.String() != result.Result {
				t.Errorf("SQL expects %v got %v", result.Result, stmt.SQL.String())
			}

			if reflect.DeepEqual(stmt.Vars, result.Vars) {
				t.Errorf("Vars expects %+v got %v", stmt.Vars, result.Vars)
			}
		})
	}
}
