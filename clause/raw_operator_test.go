package clause_test

import (
	"reflect"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestNamedExprOperatorBeforePlaceholder(t *testing.T) {
	for _, query := range []string{
		"SELECT ?::jsonb@>?::jsonb",
		"SELECT ?::jsonb<@?::jsonb",
		"SELECT to_tsvector(?)@@plainto_tsquery(?)",
		"SELECT ?::jsonb @> ?::jsonb",
		"SELECT ?::jsonb <@ ?::jsonb",
		"SELECT to_tsvector(?) @@ plainto_tsquery(?)",
	} {
		t.Run(query, func(t *testing.T) {
			vars := []interface{}{"first", "second"}
			stmt := &gorm.Statement{DB: db}
			clause.NamedExpr{SQL: query, Vars: vars}.Build(stmt)
			if stmt.SQL.String() != query {
				t.Errorf("SQL = %q, want %q", stmt.SQL.String(), query)
			}
			if !reflect.DeepEqual(stmt.Vars, vars) {
				t.Errorf("vars = %v, want %v", stmt.Vars, vars)
			}
		})
	}
}

func TestNamedExprOperatorBeforeSlicePlaceholder(t *testing.T) {
	stmt := &gorm.Statement{DB: db}
	clause.NamedExpr{
		SQL:  "SELECT ?::jsonb@>jsonb_build_array(?)",
		Vars: []interface{}{`["a","b"]`, []string{"a", "b"}},
	}.Build(stmt)
	if want := "SELECT ?::jsonb@>jsonb_build_array(?,?)"; stmt.SQL.String() != want {
		t.Errorf("SQL = %q, want %q", stmt.SQL.String(), want)
	}
	if want := []interface{}{`["a","b"]`, "a", "b"}; !reflect.DeepEqual(stmt.Vars, want) {
		t.Errorf("vars = %v, want %v", stmt.Vars, want)
	}
}
