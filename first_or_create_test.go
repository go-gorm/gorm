package gorm_test

import (
	"context"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type attrsAssignsUser struct {
	ID   uint
	Name string
	Age  int
}

// Statement.clone() used to drop the unexported attrs/assigns fields, so any
// Session-triggered clone (WithContext, PrepareStmt, SkipHooks) silently
// discarded Attrs/Assign set earlier in the chain. See BUG H01.

// openAttrsAssignsDB opens an isolated named in-memory database for the test,
// so cases sharing fixture names cannot interfere with each other.
func openAttrsAssignsDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db failed: %v", err)
	}
	if err := db.AutoMigrate(&attrsAssignsUser{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}
	return db
}

func TestAssignSurviveSessionCloneFirstOrCreate(t *testing.T) {
	db := openAttrsAssignsDB(t)
	if err := db.Create(&attrsAssignsUser{Name: "jinzhu", Age: 20}).Error; err != nil {
		t.Fatalf("create failed: %v", err)
	}

	u := attrsAssignsUser{Name: "jinzhu"}
	result := db.Where("name = ?", "jinzhu").Assign("age", 66).WithContext(context.Background()).FirstOrCreate(&u)
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if u.Age != 66 {
		t.Fatalf("expected Assign to update age to 66, got %d", u.Age)
	}
	if result.RowsAffected != 1 {
		t.Fatalf("expected RowsAffected 1, got %d", result.RowsAffected)
	}

	var stored attrsAssignsUser
	if err := db.Where("name = ?", "jinzhu").First(&stored).Error; err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if stored.Age != 66 {
		t.Fatalf("expected stored age 66, got %d", stored.Age)
	}
}

func TestAttrsSurviveSessionCloneFirstOrCreate(t *testing.T) {
	db := openAttrsAssignsDB(t)

	u := attrsAssignsUser{Name: "newbie"}
	result := db.Where("name = ?", "newbie").Attrs("age", 42).WithContext(context.Background()).FirstOrCreate(&u)
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if u.Age != 42 {
		t.Fatalf("expected Attrs to create age 42, got %d", u.Age)
	}
	if result.RowsAffected != 1 {
		t.Fatalf("expected RowsAffected 1, got %d", result.RowsAffected)
	}
}

func TestAssignSurviveSessionCloneFirstOrInit(t *testing.T) {
	db := openAttrsAssignsDB(t)
	if err := db.Create(&attrsAssignsUser{Name: "jinzhu", Age: 66}).Error; err != nil {
		t.Fatalf("create failed: %v", err)
	}

	u := attrsAssignsUser{Name: "jinzhu"}
	result := db.Where("name = ?", "jinzhu").Assign("age", 77).WithContext(context.Background()).FirstOrInit(&u)
	if result.Error != nil {
		t.Fatalf("unexpected error: %v", result.Error)
	}
	if u.Age != 77 {
		t.Fatalf("expected Assign to apply age 77 in memory, got %d", u.Age)
	}

	var stored attrsAssignsUser
	if err := db.Where("name = ?", "jinzhu").First(&stored).Error; err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if stored.Age != 66 {
		t.Fatalf("FirstOrInit must not modify the db, expected stored age 66, got %d", stored.Age)
	}
}
