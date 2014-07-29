package gorm_test

import (
	"testing"
	"time"
)

func TestDelete(t *testing.T) {
	user1, user2 := User{Name: "delete1"}, User{Name: "delete2"}
	db.Save(&user1)
	db.Save(&user2)

	if db.Delete(&user1).Error != nil {
		t.Errorf("No error should happen when delete a record")
	}

	if !db.Where("name = ?", user1.Name).First(&User{}).RecordNotFound() {
		t.Errorf("User can't be found after delete")
	}

	if db.Where("name = ?", user2.Name).First(&User{}).RecordNotFound() {
		t.Errorf("Other users that not deleted should be found-able")
	}
}

func TestInlineDelete(t *testing.T) {
	user1, user2 := User{Name: "inline_delete1"}, User{Name: "inline_delete2"}
	db.Save(&user1)
	db.Save(&user2)

	if db.Delete(&User{}, user1.Id).Error != nil {
		t.Errorf("No error should happen when delete a record")
	} else if !db.Where("name = ?", user1.Name).First(&User{}).RecordNotFound() {
		t.Errorf("User can't be found after delete")
	}

	if db.Delete(&User{}, "name = ?", user2.Name).Error != nil {
		t.Errorf("No error should happen when delete a record")
	} else if !db.Where("name = ?", user2.Name).First(&User{}).RecordNotFound() {
		t.Errorf("User can't be found after delete")
	}
}

func TestSoftDelete(t *testing.T) {
	type User struct {
		Id        int64
		Name      string
		DeletedAt time.Time
	}
	db.AutoMigrate(&User{})

	user := User{Name: "soft_delete"}
	db.Save(&user)
	db.Delete(&user)

	if db.First(&User{}, "name = ?", user.Name).Error == nil {
		t.Errorf("Can't find a soft deleted record")
	}

	if db.Unscoped().First(&User{}, "name = ?", user.Name).Error != nil {
		t.Errorf("Should be able to find soft deleted record with Unscoped")
	}

	db.Unscoped().Delete(&user)
	if !db.Unscoped().First(&User{}, "name = ?", user.Name).RecordNotFound() {
		t.Errorf("Can't find permanently deleted record")
	}
}
