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

	cases := []struct {
		name     string
		users    []updateSliceUser
		scoped   bool  // add the user's own WHERE condition
		wantRows int64 // expected RowsAffected and rows with age=100
		wantErr  error
	}{
		// H04: a trailing element with zero primary key must not discard the
		// identity IN clause built from the elements that do have identity.
		{"trailing zero PK keeps identity", []updateSliceUser{{ID: 1}, {ID: 0}}, true, 1, nil},
		// Order independence: zero-PK element first must behave the same.
		{"leading zero PK keeps identity", []updateSliceUser{{ID: 0}, {ID: 2}}, true, 1, nil},
		// Control: all elements identified -> IN (1,2) -> 2 rows.
		{"all identified", []updateSliceUser{{ID: 1}, {ID: 2}}, true, 2, nil},
		// Documented semantics: when NO element has identity the identity
		// clause is skipped and the user's own conditions scope the update
		// (mirrors the single-struct case with a zero primary key).
		{"all zero PK scoped by user where", []updateSliceUser{{ID: 0}, {ID: 0}}, true, 5, nil},
		// Safety net: no element identified and no other conditions -> refuse
		// global update instead of silently updating everything.
		{"all zero PK without where refused", []updateSliceUser{{ID: 0}, {ID: 0}}, false, 0, gorm.ErrMissingWhereClause},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			resetUpdateSliceUsers(t, db)

			tx := db.Model(&c.users)
			if c.scoped {
				tx = tx.Where("name LIKE ?", "u%")
			}
			res := tx.Updates(map[string]interface{}{"age": 100})

			if c.wantErr != nil {
				if res.Error != c.wantErr {
					t.Fatalf("expected error %v, got %v", c.wantErr, res.Error)
				}
			} else if res.Error != nil {
				t.Fatalf("updates failed: %v", res.Error)
			}
			if res.RowsAffected != c.wantRows {
				t.Fatalf("expected RowsAffected = %d, got %d", c.wantRows, res.RowsAffected)
			}
			if n := countAge100(t, db); n != c.wantRows {
				t.Fatalf("expected %d rows with age=100, got %d", c.wantRows, n)
			}
		})
	}
}
