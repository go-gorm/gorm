package tests_test

import (
	"testing"

	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
)

func TestUpdateHasOne(t *testing.T) {
	var user = *GetUser("update-has-one", Config{})

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	user.Account = Account{Number: "account-has-one-association"}

	if err := DB.Save(&user).Error; err != nil {
		t.Fatalf("errors happened when update: %v", err)
	}

	var user2 User
	DB.Preload("Account").Find(&user2, "id = ?", user.ID)
	CheckUser(t, user2, user)

	user.Account.Number += "new"
	if err := DB.Save(&user).Error; err != nil {
		t.Fatalf("errors happened when update: %v", err)
	}

	var user3 User
	DB.Preload("Account").Find(&user3, "id = ?", user.ID)
	CheckUser(t, user2, user3)

	if err := DB.Session(&gorm.Session{FullSaveAssociations: true}).Save(&user).Error; err != nil {
		t.Fatalf("errors happened when update: %v", err)
	}

	var user4 User
	DB.Preload("Account").Find(&user4, "id = ?", user.ID)
	CheckUser(t, user4, user)

	t.Run("Polymorphic", func(t *testing.T) {
		var pet = Pet{Name: "create"}

		if err := DB.Create(&pet).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		pet.Toy = Toy{Name: "Update-HasOneAssociation-Polymorphic"}

		if err := DB.Save(&pet).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		var pet2 Pet
		DB.Preload("Toy").Find(&pet2, "id = ?", pet.ID)
		CheckPet(t, pet2, pet)

		pet.Toy.Name += "new"
		if err := DB.Save(&pet).Error; err != nil {
			t.Fatalf("errors happened when update: %v", err)
		}

		var pet3 Pet
		DB.Preload("Toy").Find(&pet3, "id = ?", pet.ID)
		CheckPet(t, pet2, pet3)

		if err := DB.Session(&gorm.Session{FullSaveAssociations: true}).Save(&pet).Error; err != nil {
			t.Fatalf("errors happened when update: %v", err)
		}

		var pet4 Pet
		DB.Preload("Toy").Find(&pet4, "id = ?", pet.ID)
		CheckPet(t, pet4, pet)
	})
}
