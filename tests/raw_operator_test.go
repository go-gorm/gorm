package tests_test

import (
	"fmt"
	"testing"
)

func TestRawPostgresOperatorSpacing(t *testing.T) {
	if DB.Dialector.Name() != "postgres" {
		t.Skip("PostgreSQL JSON and full-text operators")
	}
	for _, spaced := range []bool{false, true} {
		separator := ""
		if spaced {
			separator = " "
		}
		for _, tt := range []struct {
			name, query string
			left, right interface{}
		}{
			{"contains", "SELECT ?::jsonb%s@>%s?::jsonb", `{"a":1,"b":2}`, `{"a":1}`},
			{"contained", "SELECT ?::jsonb%s<@%s?::jsonb", `{"a":1}`, `{"a":1,"b":2}`},
			{"search", "SELECT to_tsvector(?)%s@@%splainto_tsquery(?)", "green sheep", "green"},
			{"contains array", "SELECT ?::jsonb%s@>%sjsonb_build_array(?)", `["a","b"]`, []string{"a", "b"}},
		} {
			t.Run(fmt.Sprintf("%s/spaced=%v", tt.name, spaced), func(t *testing.T) {
				var matches bool
				query := fmt.Sprintf(tt.query, separator, separator)
				if err := DB.Raw(query, tt.left, tt.right).Scan(&matches).Error; err != nil {
					t.Fatal(err)
				}
				if !matches {
					t.Fatal("expected a matching PostgreSQL result")
				}
			})
		}
	}
}
