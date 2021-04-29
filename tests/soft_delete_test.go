package tests_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"regexp"
	"testing"

	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
)

func TestSoftDelete(t *testing.T) {
	user := *GetUser("SoftDelete", Config{})
	DB.Save(&user)

	var count int64
	var age uint

	if DB.Model(&User{}).Where("name = ?", user.Name).Count(&count).Error != nil || count != 1 {
		t.Errorf("Count soft deleted record, expects: %v, got: %v", 1, count)
	}

	if DB.Model(&User{}).Select("age").Where("name = ?", user.Name).Scan(&age).Error != nil || age != user.Age {
		t.Errorf("Age soft deleted record, expects: %v, got: %v", 0, age)
	}

	if err := DB.Delete(&user).Error; err != nil {
		t.Fatalf("No error should happen when soft delete user, but got %v", err)
	}

	if sql.NullTime(user.DeletedAt).Time.IsZero() {
		t.Fatalf("user's deleted at is zero")
	}

	sql := DB.Session(&gorm.Session{DryRun: true}).Delete(&user).Statement.SQL.String()
	if !regexp.MustCompile(`UPDATE .users. SET .deleted_at.=.* WHERE .users.\..id. = .* AND .users.\..deleted_at. IS NULL`).MatchString(sql) {
		t.Fatalf("invalid sql generated, got %v", sql)
	}

	if DB.First(&User{}, "name = ?", user.Name).Error == nil {
		t.Errorf("Can't find a soft deleted record")
	}

	count = 0
	if DB.Model(&User{}).Where("name = ?", user.Name).Count(&count).Error != nil || count != 0 {
		t.Errorf("Count soft deleted record, expects: %v, got: %v", 0, count)
	}

	age = 0
	if DB.Model(&User{}).Select("age").Where("name = ?", user.Name).Scan(&age).Error != nil || age != 0 {
		t.Errorf("Age soft deleted record, expects: %v, got: %v", 0, age)
	}

	if err := DB.Unscoped().First(&User{}, "name = ?", user.Name).Error; err != nil {
		t.Errorf("Should find soft deleted record with Unscoped, but got err %s", err)
	}

	count = 0
	if DB.Unscoped().Model(&User{}).Where("name = ?", user.Name).Count(&count).Error != nil || count != 1 {
		t.Errorf("Count soft deleted record, expects: %v, count: %v", 1, count)
	}

	age = 0
	if DB.Unscoped().Model(&User{}).Select("age").Where("name = ?", user.Name).Scan(&age).Error != nil || age != user.Age {
		t.Errorf("Age soft deleted record, expects: %v, got: %v", 0, age)
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
