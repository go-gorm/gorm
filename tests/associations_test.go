package tests_test

import (
	"testing"

	. "github.com/jinzhu/gorm/tests"
)

func AssertAssociationCount(t *testing.T, data interface{}, name string, result int64, reason string) {
	if count := DB.Model(data).Association(name).Count(); count != result {
		t.Fatalf("invalid %v count %v, expects: %v got %v", name, reason, result, count)
	}

	var newUser User
	if user, ok := data.(User); ok {
		DB.Find(&newUser, "id = ?", user.ID)
	} else if user, ok := data.(*User); ok {
		DB.Find(&newUser, "id = ?", user.ID)
	}

	if newUser.ID != 0 {
		if count := DB.Model(&newUser).Association(name).Count(); count != result {
			t.Fatalf("invalid %v count %v, expects: %v got %v", name, reason, result, count)
		}
	}
}

func TestInvalidAssociation(t *testing.T) {
	var user = *GetUser("invalid", Config{Company: true, Manager: true})
	if err := DB.Model(&user).Association("Invalid").Find(&user.Company).Error; err == nil {
		t.Fatalf("should return errors for invalid association, but got nil")
	}
}
