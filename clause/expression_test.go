package clause_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/clause"
	"github.com/jinzhu/gorm/schema"
	"github.com/jinzhu/gorm/tests"
)

func TestExpr(t *testing.T) {
	results := []struct {
		SQL    string
		Result string
		Vars   []interface{}
	}{{
		SQL:    "create table ? (? ?, ? ?)",
		Vars:   []interface{}{clause.Table{Name: "users"}, clause.Column{Name: "id"}, clause.Expr{SQL: "int"}, clause.Column{Name: "name"}, clause.Expr{SQL: "text"}},
		Result: "create table `users` (`id` int, `name` text)",
	}}

	for idx, result := range results {
		t.Run(fmt.Sprintf("case #%v", idx), func(t *testing.T) {
			user, _, _ := schema.Parse(&tests.User{}, &sync.Map{}, db.NamingStrategy)
			stmt := &gorm.Statement{DB: db, Table: user.Table, Schema: user, Clauses: map[string]clause.Clause{}}
			clause.Expr{SQL: result.SQL, Vars: result.Vars}.Build(stmt)
			if stmt.SQL.String() != result.Result {
				t.Errorf("generated SQL is not equal, expects %v, but got %v", result.Result, stmt.SQL.String())
			}
		})
	}
}
