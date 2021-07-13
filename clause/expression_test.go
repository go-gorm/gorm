package clause_test

import (
	"database/sql"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils/tests"
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
			user, _ := schema.Parse(&tests.User{}, &sync.Map{}, db.NamingStrategy)
			stmt := &gorm.Statement{DB: db, Table: user.Table, Schema: user, Clauses: map[string]clause.Clause{}}
			clause.Expr{SQL: result.SQL, Vars: result.Vars}.Build(stmt)
			if stmt.SQL.String() != result.Result {
				t.Errorf("generated SQL is not equal, expects %v, but got %v", result.Result, stmt.SQL.String())
			}
		})
	}
}

func TestNamedExpr(t *testing.T) {
	type Base struct {
		Name2 string
	}

	type NamedArgument struct {
		Name1 string
		Base
	}

	results := []struct {
		SQL          string
		Result       string
		Vars         []interface{}
		ExpectedVars []interface{}
	}{{
		SQL:    "create table ? (? ?, ? ?)",
		Vars:   []interface{}{clause.Table{Name: "users"}, clause.Column{Name: "id"}, clause.Expr{SQL: "int"}, clause.Column{Name: "name"}, clause.Expr{SQL: "text"}},
		Result: "create table `users` (`id` int, `name` text)",
	}, {
		SQL:          "name1 = @name AND name2 = @name",
		Vars:         []interface{}{sql.Named("name", "jinzhu")},
		Result:       "name1 = ? AND name2 = ?",
		ExpectedVars: []interface{}{"jinzhu", "jinzhu"},
	}, {
		SQL:          "name1 = @name AND name2 = @@name",
		Vars:         []interface{}{map[string]interface{}{"name": "jinzhu"}},
		Result:       "name1 = ? AND name2 = @@name",
		ExpectedVars: []interface{}{"jinzhu"},
	}, {
		SQL:          "name1 = @name1 AND name2 = @name2 AND name3 = @name1",
		Vars:         []interface{}{sql.Named("name1", "jinzhu"), sql.Named("name2", "jinzhu2")},
		Result:       "name1 = ? AND name2 = ? AND name3 = ?",
		ExpectedVars: []interface{}{"jinzhu", "jinzhu2", "jinzhu"},
	}, {
		SQL:          "name1 = @name1 AND name2 = @name2 AND name3 = @name1",
		Vars:         []interface{}{map[string]interface{}{"name1": "jinzhu", "name2": "jinzhu2"}},
		Result:       "name1 = ? AND name2 = ? AND name3 = ?",
		ExpectedVars: []interface{}{"jinzhu", "jinzhu2", "jinzhu"},
	}, {
		SQL:          "@@test AND name1 = @name1 AND name2 = @name2 AND name3 = @name1 @notexist",
		Vars:         []interface{}{sql.Named("name1", "jinzhu"), sql.Named("name2", "jinzhu2")},
		Result:       "@@test AND name1 = ? AND name2 = ? AND name3 = ? @notexist",
		ExpectedVars: []interface{}{"jinzhu", "jinzhu2", "jinzhu"},
	}, {
		SQL:          "@@test AND name1 = @Name1 AND name2 = @Name2 AND name3 = @Name1 @notexist",
		Vars:         []interface{}{NamedArgument{Name1: "jinzhu", Base: Base{Name2: "jinzhu2"}}},
		Result:       "@@test AND name1 = ? AND name2 = ? AND name3 = ? @notexist",
		ExpectedVars: []interface{}{"jinzhu", "jinzhu2", "jinzhu"},
	}, {
		SQL:    "create table ? (? ?, ? ?)",
		Vars:   []interface{}{},
		Result: "create table ? (? ?, ? ?)",
	}}

	for idx, result := range results {
		t.Run(fmt.Sprintf("case #%v", idx), func(t *testing.T) {
			user, _ := schema.Parse(&tests.User{}, &sync.Map{}, db.NamingStrategy)
			stmt := &gorm.Statement{DB: db, Table: user.Table, Schema: user, Clauses: map[string]clause.Clause{}}
			clause.NamedExpr{SQL: result.SQL, Vars: result.Vars}.Build(stmt)
			if stmt.SQL.String() != result.Result {
				t.Errorf("generated SQL is not equal, expects %v, but got %v", result.Result, stmt.SQL.String())
			}

			if !reflect.DeepEqual(result.ExpectedVars, stmt.Vars) {
				t.Errorf("generated vars is not equal, expects %v, but got %v", result.ExpectedVars, stmt.Vars)
			}
		})
	}
}

func TestExpression(t *testing.T) {
	column := "column-name"
	results := []struct {
		Expressions  []clause.Expression
		ExpectedVars []interface{}
		Result       string
	}{{
		Expressions: []clause.Expression{
			clause.Eq{Column: column, Value: "column-value"},
		},
		ExpectedVars: []interface{}{"column-value"},
		Result:       "`column-name` = ?",
	}, {
		Expressions: []clause.Expression{
			clause.Eq{Column: column, Value: nil},
			clause.Eq{Column: column, Value: (*string)(nil)},
			clause.Eq{Column: column, Value: (*int)(nil)},
			clause.Eq{Column: column, Value: (*bool)(nil)},
			clause.Eq{Column: column, Value: (interface{})(nil)},
			clause.Eq{Column: column, Value: sql.NullString{String: "", Valid: false}},
		},
		Result: "`column-name` IS NULL",
	}, {
		Expressions: []clause.Expression{
			clause.Neq{Column: column, Value: "column-value"},
		},
		ExpectedVars: []interface{}{"column-value"},
		Result:       "`column-name` <> ?",
	}, {
		Expressions: []clause.Expression{
			clause.Neq{Column: column, Value: nil},
			clause.Neq{Column: column, Value: (*string)(nil)},
			clause.Neq{Column: column, Value: (*int)(nil)},
			clause.Neq{Column: column, Value: (*bool)(nil)},
			clause.Neq{Column: column, Value: (interface{})(nil)},
		},
		Result: "`column-name` IS NOT NULL",
	}, {
		Expressions: []clause.Expression{
			clause.Eq{Column: column, Value: []string{"a", "b"}},
		},
		ExpectedVars: []interface{}{"a", "b"},
		Result:       "`column-name` IN (?,?)",
	}, {
		Expressions: []clause.Expression{
			clause.Neq{Column: column, Value: []string{"a", "b"}},
		},
		ExpectedVars: []interface{}{"a", "b"},
		Result:       "`column-name` NOT IN (?,?)",
	}}

	for idx, result := range results {
		for idy, expression := range result.Expressions {
			t.Run(fmt.Sprintf("case #%v.%v", idx, idy), func(t *testing.T) {
				user, _ := schema.Parse(&tests.User{}, &sync.Map{}, db.NamingStrategy)
				stmt := &gorm.Statement{DB: db, Table: user.Table, Schema: user, Clauses: map[string]clause.Clause{}}
				expression.Build(stmt)
				if stmt.SQL.String() != result.Result {
					t.Errorf("generated SQL is not equal, expects %v, but got %v", result.Result, stmt.SQL.String())
				}

				if !reflect.DeepEqual(result.ExpectedVars, stmt.Vars) {
					t.Errorf("generated vars is not equal, expects %v, but got %v", result.ExpectedVars, stmt.Vars)
				}
			})
		}
	}
}
