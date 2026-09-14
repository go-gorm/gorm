package gorm_test

import (
	"fmt"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type updateSliceUser struct {
	ID   uint `gorm:"primaryKey"`
	Name string
	Age  int
}

func resetUpdateSliceUsers(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec("DELETE FROM update_slice_users").Error; err != nil {
		t.Fatalf("clean table failed: %v", err)
	}
	for i := 1; i <= 5; i++ {
		if err := db.Create(&updateSliceUser{ID: uint(i), Name: fmt.Sprintf("u%d", i), Age: i}).Error; err != nil {
			t.Fatalf("seed failed: %v", err)
		}
	}
}

func countAge100(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var n int64
	if err := db.Model(&updateSliceUser{}).Where("age = ?", 100).Count(&n).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	return n
}

func TestUpdatesSliceIdentityClause(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db failed: %v", err)
	}
	if err := db.AutoMigrate(&updateSliceUser{}); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	// H04: a trailing element with zero primary key must not discard the
	// identity IN clause built from the elements that do have identity.
	resetUpdateSliceUsers(t, db)
	users := []updateSliceUser{{ID: 1}, {ID: 0}}
	res := db.Model(&users).Where("name LIKE ?", "u%").Updates(map[string]interface{}{"age": 100})
	if res.Error != nil {
		t.Fatalf("updates failed: %v", res.Error)
	}
	if res.RowsAffected != 1 {
		t.Fatalf("expected only the identified row to be updated, RowsAffected = %d", res.RowsAffected)
	}
	if n := countAge100(t, db); n != 1 {
		t.Fatalf("expected 1 row with age=100, got %d", n)
	}

	// Order independence: zero-PK element first must behave the same.
	resetUpdateSliceUsers(t, db)
	users = []updateSliceUser{{ID: 0}, {ID: 2}}
	res = db.Model(&users).Where("name LIKE ?", "u%").Updates(map[string]interface{}{"age": 100})
	if res.Error != nil {
		t.Fatalf("updates failed: %v", res.Error)
	}
	if res.RowsAffected != 1 || countAge100(t, db) != 1 {
		t.Fatalf("expected only row 2 updated, RowsAffected = %d, age100 = %d", res.RowsAffected, countAge100(t, db))
	}

	// Control: all elements identified -> IN (1,2) -> 2 rows.
	resetUpdateSliceUsers(t, db)
	users = []updateSliceUser{{ID: 1}, {ID: 2}}
	res = db.Model(&users).Where("name LIKE ?", "u%").Updates(map[string]interface{}{"age": 100})
	if res.Error != nil {
		t.Fatalf("updates failed: %v", res.Error)
	}
	if res.RowsAffected != 2 || countAge100(t, db) != 2 {
		t.Fatalf("expected 2 rows updated, RowsAffected = %d, age100 = %d", res.RowsAffected, countAge100(t, db))
	}

	// Documented semantics: elements whose primary keys are all zero carry no
	// identity, so when NO element has identity the identity clause is skipped
	// and the user's own conditions scope the update (mirrors the single-struct
	// case with a zero primary key).
	resetUpdateSliceUsers(t, db)
	users = []updateSliceUser{{ID: 0}, {ID: 0}}
	res = db.Model(&users).Where("name LIKE ?", "u%").Updates(map[string]interface{}{"age": 100})
	if res.Error != nil {
		t.Fatalf("updates failed: %v", res.Error)
	}
	if res.RowsAffected != 5 {
		t.Fatalf("expected explicit WHERE to scope the update to 5 rows, RowsAffected = %d", res.RowsAffected)
	}

	// Safety net: no element identified and no other conditions -> refuse
	// global update instead of silently updating everything.
	resetUpdateSliceUsers(t, db)
	users = []updateSliceUser{{ID: 0}, {ID: 0}}
	res = db.Model(&users).Updates(map[string]interface{}{"age": 100})
	if res.Error != gorm.ErrMissingWhereClause {
		t.Fatalf("expected ErrMissingWhereClause, got %v", res.Error)
	}
	if res.RowsAffected != 0 || countAge100(t, db) != 0 {
		t.Fatalf("expected no rows updated, RowsAffected = %d, age100 = %d", res.RowsAffected, countAge100(t, db))
	}
}
