package tests

import (
	"fmt"
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

	TestCreateAssociations(t, db)
}

func TestCreateAssociations(t *testing.T, db *gorm.DB) {
	db.Migrator().DropTable(&Company{})
	db.Migrator().AutoMigrate(&Company{})

	t.Run("Create-BelongsToAssociation", func(t *testing.T) {
		var user = User{
			Name:     "create",
			Age:      18,
			Birthday: Now(),
			Company:  Company{Name: "company-belongs-to-association"},
			Manager:  &User{Name: "manager-belongs-to-association"},
		}

		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		if user.CompanyID == nil {
			t.Errorf("Failed to create belongs to association - Company")
		} else {
			var company Company
			db.First(&company, "id = ?", *user.CompanyID)
			if company.Name != "company-belongs-to-association" {
				t.Errorf("Failed to query saved belongs to association - Company")
			}
		}

		if user.ManagerID == nil {
			t.Errorf("Failed to create belongs to association - Manager")
		} else {
			var manager User
			db.First(&manager, "id = ?", *user.ManagerID)
			if manager.Name != "manager-belongs-to-association" {
				t.Errorf("Failed to query saved belongs to association - Manager")
			}
		}
	})

	t.Run("Create-HasOneAssociation", func(t *testing.T) {
		var user = User{
			Name:     "create",
			Age:      18,
			Birthday: Now(),
			Account:  Account{Number: "account-has-one-association"},
		}

		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		if user.Account.ID == 0 {
			t.Errorf("Failed to create has one association - Account")
		} else if user.Account.UserID.Int64 != int64(user.ID) {
			t.Errorf("Failed to create has one association - Account")
		} else {
			var account Account
			db.First(&account, "id = ?", user.Account.ID)
			if user.Account.Number != "account-has-one-association" {
				t.Errorf("Failed to query saved has one association - Account")
			}
		}
	})

	t.Run("Create-HasOneAssociation-Polymorphic", func(t *testing.T) {
		var pet = Pet{
			Name: "create",
			Toy:  Toy{Name: "Create-HasOneAssociation-Polymorphic"},
		}

		if err := db.Create(&pet).Error; err != nil {
			t.Fatalf("errors happened when create: %v", err)
		}

		if pet.Toy.OwnerID != fmt.Sprint(pet.ID) || pet.Toy.OwnerType != "pets" {
			t.Errorf("Failed to create polymorphic has one association - toy owner id %v, owner type %v", pet.Toy.OwnerID, pet.Toy.OwnerType)
		} else {
			var toy Toy
			db.First(&toy, "owner_id = ? and owner_type = ?", pet.Toy.OwnerID, pet.Toy.OwnerType)
			if toy.Name != "Create-HasOneAssociation-Polymorphic" {
				t.Errorf("Failed to query saved polymorphic has one association")
			}
		}
	})
}
