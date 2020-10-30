package tests_test

import (
	"encoding/json"
	"errors"
	"testing"

	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
)

func TestSoftDelete(t *testing.T) {
	user := *GetUser("SoftDelete", Config{})
	DB.Save(&user)

	var count int64
	if DB.Model(&User{}).Where("name = ?", user.Name).Count(&count).Error != nil || count != 1 {
		t.Errorf("Count soft deleted record, expects: %v, got: %v", 1, count)
	}

	if err := DB.Delete(&user).Error; err != nil {
		t.Fatalf("No error should happen when soft delete user, but got %v", err)
	}

	if DB.First(&User{}, "name = ?", user.Name).Error == nil {
		t.Errorf("Can't find a soft deleted record")
	}

	if DB.Model(&User{}).Where("name = ?", user.Name).Count(&count).Error != nil || count != 0 {
		t.Errorf("Count soft deleted record, expects: %v, got: %v", 0, count)
	}

	if err := DB.Unscoped().First(&User{}, "name = ?", user.Name).Error; err != nil {
		t.Errorf("Should find soft deleted record with Unscoped, but got err %s", err)
	}

	if DB.Unscoped().Model(&User{}).Where("name = ?", user.Name).Count(&count).Error != nil || count != 1 {
		t.Errorf("Count soft deleted record, expects: %v, count: %v", 1, count)
	}

	DB.Unscoped().Delete(&user)
	if err := DB.Unscoped().First(&User{}, "name = ?", user.Name).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("Can't find permanently deleted record")
	}
}

func TestDeletedAtUnMarshal(t *testing.T) {
	expected := &gorm.Model{}
	b, _ := json.Marshal(expected)

	result := &gorm.Model{}
	_ = json.Unmarshal(b, result)
	if result.DeletedAt != expected.DeletedAt {
		t.Errorf("Failed, result.DeletedAt: %v is not same as expected.DeletedAt: %v", result.DeletedAt, expected.DeletedAt)
	}
}
