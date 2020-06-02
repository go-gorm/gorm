package tests_test

import (
	"testing"

	. "gorm.io/gorm/tests"
)

func TestSoftDelete(t *testing.T) {
	user := *GetUser("SoftDelete", Config{})
	DB.Save(&user)
	if err := DB.Delete(&user).Error; err != nil {
		t.Fatalf("No error should happen when soft delete user, but got %v", err)
	}

	if DB.First(&User{}, "name = ?", user.Name).Error == nil {
		t.Errorf("Can't find a soft deleted record")
	}

	if err := DB.Unscoped().First(&User{}, "name = ?", user.Name).Error; err != nil {
		t.Errorf("Should find soft deleted record with Unscoped, but got err %s", err)
	}

	DB.Unscoped().Delete(&user)
	if !DB.Unscoped().First(&User{}, "name = ?", user.Name).RecordNotFound() {
		t.Errorf("Can't find permanently deleted record")
	}
}
