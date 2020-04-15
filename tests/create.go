package tests

import (
	"testing"

	"github.com/jinzhu/gorm"
)

func TestCreate(t *testing.T, db *gorm.DB) {
	db.Migrator().DropTable(&User{})
	db.AutoMigrate(&User{})

	t.Run("Create", func(t *testing.T) {
		var user = User{
			Name:     "create",
			Age:      18,
			Birthday: Now(),
		}

		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		if user.ID == 0 {
			t.Errorf("user's primary key should has value after create, got : %v", user.ID)
		}

		if user.CreatedAt.IsZero() {
			t.Errorf("user's created at should be not zero")
		}

		if user.UpdatedAt.IsZero() {
			t.Errorf("user's updated at should be not zero")
		}

		var newUser User
		if err := db.Where("id = ?", user.ID).First(&newUser).Error; err != nil {
			t.Errorf("errors happened when query: %v", err)
		} else {
			AssertObjEqual(t, newUser, user, "Name", "Age", "Birthday")
		}
	})
}
