package tests_test

import (
	"testing"

	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
)

func TestUpdateBelongsTo(t *testing.T) {
	var user = *GetUser("update-belongs-to", Config{})

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	user.Company = Company{Name: "company-belongs-to-association"}
	user.Manager = &User{Name: "manager-belongs-to-association"}
	if err := DB.Save(&user).Error; err != nil {
		t.Fatalf("errors happened when update: %v", err)
	}

	var user2 User
	DB.Preload("Company").Preload("Manager").Find(&user2, "id = ?", user.ID)
	CheckUser(t, user2, user)

	user.Company.Name += "new"
	user.Manager.Name += "new"
	if err := DB.Save(&user).Error; err != nil {
		t.Fatalf("errors happened when update: %v", err)
	}

	var user3 User
	DB.Preload("Company").Preload("Manager").Find(&user3, "id = ?", user.ID)
	CheckUser(t, user2, user3)

	if err := DB.Session(&gorm.Session{FullSaveAssociations: true}).Save(&user).Error; err != nil {
		t.Fatalf("errors happened when update: %v", err)
	}

	var user4 User
	DB.Preload("Company").Preload("Manager").Find(&user4, "id = ?", user.ID)
	CheckUser(t, user4, user)
}
