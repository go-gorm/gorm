package gorm_test

import (
	"testing"
)

func TestDdlErrors(t *testing.T) {
	var err error

	if err = DB.Close(); err != nil {
		t.Errorf("Closing DDL test db connection err=%s", err)
	}
	defer func() {
		// Reopen DB connection.
		if DB, err = OpenTestConnection(); err != nil {
			t.Fatalf("Failed re-opening db connection: %s", err)
		}
	}()

	DB.HasTable("foobarbaz")
	if DB.Error == nil {
		t.Errorf("Expected operation on closed db to produce an error, but err was nil")
	}
}
