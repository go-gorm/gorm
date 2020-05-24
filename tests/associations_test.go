package tests_test

import (
	"testing"

	. "github.com/jinzhu/gorm/tests"
)

func TestAssociationForBelongsTo(t *testing.T) {
	var user = *GetUser("belongs-to", Config{Company: true, Manager: true})

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	CheckUser(t, user, user)

	var user2 User
	DB.Find(&user2, "id = ?", user.ID)
	DB.Model(&user2).Association("Company").Find(&user2.Company)
	user2.Manager = &User{}
	DB.Model(&user2).Association("Manager").Find(user2.Manager)
	CheckUser(t, user2, user)

	if count := DB.Model(&user).Association("Company").Count(); count != 1 {
		t.Errorf("invalid company count, got %v", count)
	}

	if count := DB.Model(&user).Association("Manager").Count(); count != 1 {
		t.Errorf("invalid manager count, got %v", count)
	}
}
