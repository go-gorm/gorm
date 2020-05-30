package tests_test

import (
	"testing"

	. "github.com/jinzhu/gorm/tests"
)

func TestUpdateHasManyAssociations(t *testing.T) {
	var user = *GetUser("update-has-many", Config{})

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	user.Pets = []*Pet{{Name: "pet1"}, {Name: "pet2"}}
	if err := DB.Save(&user).Error; err != nil {
		t.Fatalf("errors happened when update: %v", err)
	}

	var user2 User
	DB.Preload("Pets").Find(&user2, "id = ?", user.ID)
	CheckUser(t, user2, user)

	t.Run("Polymorphic", func(t *testing.T) {
		var user = *GetUser("update-has-many", Config{})

		if err := DB.Create(&user).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		user.Toys = []Toy{{Name: "toy1"}, {Name: "toy2"}}
		if err := DB.Save(&user).Error; err != nil {
			t.Fatalf("errors happened when update: %v", err)
		}

		var user2 User
		DB.Preload("Toys").Find(&user2, "id = ?", user.ID)
		CheckUser(t, user2, user)
	})
}
