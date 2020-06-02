package tests_test

import (
	"errors"
	"testing"

	"gorm.io/gorm"
	. "gorm.io/gorm/tests"
)

func TestDelete(t *testing.T) {
	var users = []User{*GetUser("delete", Config{}), *GetUser("delete", Config{}), *GetUser("delete", Config{})}

	if err := DB.Create(&users).Error; err != nil {
		t.Errorf("errors happened when create: %v", err)
	}

	for _, user := range users {
		if user.ID == 0 {
			t.Fatalf("user's primary key should has value after create, got : %v", user.ID)
		}
	}

	if err := DB.Delete(&users[1]).Error; err != nil {
		t.Errorf("errors happened when delete: %v", err)
	}

	var result User
	if err := DB.Where("id = ?", users[1].ID).First(&result).Error; err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("should returns record not found error, but got %v", err)
	}

	for _, user := range []User{users[0], users[2]} {
		result = User{}
		if err := DB.Where("id = ?", user.ID).First(&result).Error; err != nil {
			t.Errorf("no error should returns when query %v, but got %v", user.ID, err)
		}
	}

	for _, user := range []User{users[0], users[2]} {
		result = User{}
		if err := DB.Where("id = ?", user.ID).First(&result).Error; err != nil {
			t.Errorf("no error should returns when query %v, but got %v", user.ID, err)
		}
	}
}

func TestInlineCondDelete(t *testing.T) {
	user1 := *GetUser("inline_delete_1", Config{})
	user2 := *GetUser("inline_delete_2", Config{})
	DB.Save(&user1).Save(&user2)

	if DB.Delete(&User{}, user1.ID).Error != nil {
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

func TestBlockGlobalDelete(t *testing.T) {
	if err := DB.Delete(&User{}).Error; err == nil || !errors.Is(err, gorm.ErrMissingWhereClause) {
		t.Errorf("should returns missing WHERE clause while deleting error")
	}
}
