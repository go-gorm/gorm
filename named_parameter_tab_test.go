package gorm_test

import (
	"database/sql"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRawNamedParameterTab(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	conn, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Errorf("close SQLite connection: %v", err)
		}
	})
	for _, tc := range []struct{ name, query string }{
		{"before_alias", "SELECT @value\tAS result"},
		{"trailing_tab", "SELECT @value\t"},
		{"space_control", "SELECT @value AS result"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got sql.NullString
			if err := db.Raw(tc.query, sql.Named("value", "bound")).Row().Scan(&got); err != nil {
				t.Fatal(err)
			}
			if !got.Valid || got.String != "bound" {
				t.Errorf("got %#v, want a non-NULL bound value", got)
			}
		})
	}
}
