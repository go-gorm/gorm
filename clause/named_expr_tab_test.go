package clause_test

import (
	"database/sql"
	"reflect"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestNamedExprTabDelimiter(t *testing.T) {
	cases := []struct {
		name, query, want string
		args              []interface{}
		vars              []interface{}
	}{
		{"named_arg", "name = @name\tAND age = @age", "name = ?\tAND age = ?", []interface{}{sql.Named("name", "alice"), sql.Named("age", 21)}, []interface{}{"alice", 21}},
		{"map", "name = @name\tAND age = @age", "name = ?\tAND age = ?", []interface{}{map[string]interface{}{"name": "alice", "age": 21}}, []interface{}{"alice", 21}},
		{"struct", "name = @Name\tAND age = @Age", "name = ?\tAND age = ?", []interface{}{struct {
			Name string
			Age  int
		}{"alice", 21}}, []interface{}{"alice", 21}},
		{"trailing_tab", "name = @name\t", "name = ?\t", []interface{}{sql.Named("name", "alice")}, []interface{}{"alice"}},
		{"repeated_tabs", "name = @name\t\tAND other = @name", "name = ?\t\tAND other = ?", []interface{}{sql.Named("name", "alice")}, []interface{}{"alice", "alice"}},
		{"tab_before_newline", "name = @name\t\nAND age = @age", "name = ?\t\nAND age = ?", []interface{}{sql.Named("name", "alice"), sql.Named("age", 21)}, []interface{}{"alice", 21}},
		{"slice", "name IN (@names\t)", "name IN (?,?\t)", []interface{}{sql.Named("names", []string{"alice", "bob"})}, []interface{}{"alice", "bob"}},
		{"empty_slice", "name IN (@names\t)", "name IN (?\t)", []interface{}{sql.Named("names", []string{})}, []interface{}{nil}},
		{"unknown_name", "name = @unknown\tAND age = @age", "name = @unknown\tAND age = ?", []interface{}{sql.Named("age", 21)}, []interface{}{21}},
		{"no_tab", "name = @name AND age = @age", "name = ? AND age = ?", []interface{}{sql.Named("name", "alice"), sql.Named("age", 21)}, []interface{}{"alice", 21}},
		{"positional", "name = ?\tAND age = ?", "name = ?\tAND age = ?", []interface{}{"alice", 21}, []interface{}{"alice", 21}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stmt := &gorm.Statement{DB: db}
			clause.NamedExpr{SQL: tc.query, Vars: tc.args}.Build(stmt)
			if got := stmt.SQL.String(); got != tc.want {
				t.Errorf("SQL: got %q, want %q", got, tc.want)
			}
			if !reflect.DeepEqual(stmt.Vars, tc.vars) {
				t.Errorf("vars: got %#v, want %#v", stmt.Vars, tc.vars)
			}
		})
	}
}
