package gorm_test

import (
	"testing"
	"time"
)

func TestDelete(t *testing.T) {
	user1, user2 := User{Name: "delete1"}, User{Name: "delete2"}
	DB.Save(&user1)
	DB.Save(&user2)

	if err := DB.Delete(&user1).Error; err != nil {
		t.Errorf("No error should happen when delete a record, err=%s", err)
	}

	if !DB.Where("name = ?", user1.Name).First(&User{}).RecordNotFound() {
		t.Errorf("User can't be found after delete")
	}

	if DB.Where("name = ?", user2.Name).First(&User{}).RecordNotFound() {
		t.Errorf("Other users that not deleted should be found-able")
	}
}

func TestInlineDelete(t *testing.T) {
	user1, user2 := User{Name: "inline_delete1"}, User{Name: "inline_delete2"}
	DB.Save(&user1)
	DB.Save(&user2)

	if DB.Delete(&User{}, user1.Id).Error != nil {
		t.Errorf("No error should happen when delete a record")
	} else if !DB.Where("name = ?", user1.Name).First(&User{}).RecordNotFound() {
		t.Errorf("User can't be found after delete")
	}

	if err := DB.Delete(&User{}, "name = ?", user2.Name).Error; err != nil {
		t.Errorf("No error should happen when delete a record, err=%s", err)
	} else if !DB.Where("name = ?", user2.Name).First(&User{}).RecordNotFound() {
		t.Errorf("User can't be found after delete")
	}
}

func TestSoftDelete(t *testing.T) {
	type User struct {
		Id        int64
		Name      string
		DeletedAt time.Time
	}
	DB.AutoMigrate(&User{})

	user := User{Name: "soft_delete"}
	DB.Save(&user)
	DB.Delete(&user)

	if DB.First(&User{}, "name = ?", user.Name).Error == nil {
		t.Errorf("Can't find a soft deleted record")
	}

	if err := DB.Unscoped().First(&User{}, "name = ?", user.Name).Error; err != nil {
		t.Errorf("Should be able to find soft deleted record with Unscoped, but err=%s", err)
	}

	DB.Unscoped().Delete(&user)
	if !DB.Unscoped().First(&User{}, "name = ?", user.Name).RecordNotFound() {
		t.Errorf("Can't find permanently deleted record")
	}
}
