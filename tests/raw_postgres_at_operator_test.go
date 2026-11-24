package tests_test

import (
	"database/sql"
	"testing"
)

func TestRawPostgresAtOperators(t *testing.T) {
	if DB.Dialector.Name() != "postgres" {
		t.Skip("requires PostgreSQL")
	}

	const (
		document = `{"a":[1,2],"b":true}`
		subset   = `{"b":true}`
		path     = `$.a[*] ? (@ > 1)`
	)

	for _, tt := range []struct {
		name  string
		query string
		args  []interface{}
	}{
		{"contains, spaced", "SELECT ?::jsonb @> ?::jsonb", []interface{}{document, subset}},
		{"contains, compact", "SELECT ?::jsonb@>?::jsonb", []interface{}{document, subset}},
		{"contained by, compact", "SELECT ?::jsonb<@?::jsonb", []interface{}{subset, document}},
		{"full text match, compact", "SELECT to_tsvector(?)@@plainto_tsquery(?)", []interface{}{"green sheep", "green"}},
		{"contains array, compact", "SELECT ?::jsonb@>jsonb_build_array(?)", []interface{}{`["a","b"]`, []string{"a", "b"}}},
		{"path exists, spaced", "SELECT ?::jsonb @? ?::jsonpath", []interface{}{document, path}},
		{"path exists, compact", "SELECT ?::jsonb@??::jsonpath", []interface{}{document, path}},
		{"path exists, named path", "SELECT ?::jsonb@?@path", []interface{}{document, sql.Named("path", path)}},
		{"less than a named parameter", "SELECT 1<@limit", []interface{}{sql.Named("limit", 2)}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var matches bool
			if err := DB.Raw(tt.query, tt.args...).Row().Scan(&matches); err != nil {
				t.Fatal(err)
			}
			if !matches {
				t.Fatal("expected the expression to evaluate to true")
			}
		})
	}
}
