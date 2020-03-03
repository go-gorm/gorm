package tests

import (
	"testing"
	"time"

	"github.com/jinzhu/gorm"
)

func Now() *time.Time {
	now := time.Now()
	return &now
}

func RunTestsSuit(t *testing.T, db *gorm.DB) {
	TestCreate(t, db)
}

func TestCreate(t *testing.T, db *gorm.DB) {
	db.AutoMigrate(&User{})

	t.Run("Create", func(t *testing.T) {
		var user = User{
			Name:     "create",
			Age:      18,
			Birthday: Now(),
		}

		if err := db.Create(&user).Error; err != nil {
			t.Errorf("errors happened when create: %v", err)
		}

		if user.ID == 0 {
			t.Errorf("user's primary key should has value after create, got : %v", user.ID)
		}

		var newUser User
		if err := db.Where("id = ?", user.ID).First(&newUser).Error; err != nil {
			t.Errorf("errors happened when query: %v", err)
		} else {
			AssertEqual(t, newUser, user, "Name", "Age", "Birthday")
		}
	})
}
