package tests

import (
	"errors"
	"testing"

	"github.com/jinzhu/gorm"
)

func TestDelete(t *testing.T, db *gorm.DB) {
	db.Migrator().DropTable(&User{})
	db.AutoMigrate(&User{})

	t.Run("Delete", func(t *testing.T) {
		var users = []User{{
			Name:     "find",
			Age:      1,
			Birthday: Now(),
		}, {
			Name:     "find",
			Age:      2,
			Birthday: Now(),
		}, {
			Name:     "find",
			Age:      3,
			Birthday: Now(),
		}}

		if err := db.Create(&users).Error; err != nil {
			t.Errorf("errors happened when create: %v", err)
		}

		for _, user := range users {
			if user.ID == 0 {
				t.Fatalf("user's primary key should has value after create, got : %v", user.ID)
			}
		}

		if err := db.Delete(&users[1]).Error; err != nil {
			t.Errorf("errors happened when delete: %v", err)
		}

		var result User
		if err := db.Where("id = ?", users[1].ID).First(&result).Error; err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Errorf("should returns record not found error, but got %v", err)
		}

		for _, user := range []User{users[0], users[2]} {
			if err := db.Where("id = ?", user.ID).First(&result).Error; err != nil {
				t.Errorf("no error should returns when query %v, but got %v", user.ID, err)
			}
		}

		if err := db.Delete(&User{}).Error; err == nil || !errors.Is(err, gorm.ErrMissingWhereClause) {
			t.Errorf("should returns missing WHERE clause while deleting error")
		}

		for _, user := range []User{users[0], users[2]} {
			if err := db.Where("id = ?", user.ID).First(&result).Error; err != nil {
				t.Errorf("no error should returns when query %v, but got %v", user.ID, err)
			}
		}
	})
}
