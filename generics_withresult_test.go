package gorm_test

import (
	"context"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type withResultUser struct {
	ID   uint
	Name string
}

func TestWithResultIncludesError(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db failed: %v", err)
	}
	if err := db.AutoMigrate(&withResultUser{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	ctx := context.Background()

	u := withResultUser{Name: "with-result-ok"}
	r := gorm.WithResult()
	if err := gorm.G[withResultUser](db, r).Create(ctx, &u); err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if u.ID == 0 {
		t.Fatalf("expected ID to be set")
	}
	if r.Error != nil {
		t.Fatalf("unexpected result error: %v", r.Error)
	}
	if r.RowsAffected <= 0 {
		t.Fatalf("expected RowsAffected > 0, got %d", r.RowsAffected)
	}

	u2 := withResultUser{Name: "with-result-fail"}
	r2 := gorm.WithResult()
	if err := gorm.G[withResultUser](db, r2).Table("does_not_exist").Create(ctx, &u2); err == nil {
		t.Fatalf("expected error for missing table, got nil")
	}
	if r2.Error == nil {
		t.Fatalf("expected result error to be set for missing table")
	}
}

