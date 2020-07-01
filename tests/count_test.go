package tests_test

import (
	"fmt"
	"testing"

	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
)

func TestCount(t *testing.T) {
	var (
		user1                 = *GetUser("count-1", Config{})
		user2                 = *GetUser("count-2", Config{})
		user3                 = *GetUser("count-3", Config{})
		users                 []User
		count, count1, count2 int64
	)

	DB.Save(&user1).Save(&user2).Save(&user3)

	if err := DB.Where("name = ?", user1.Name).Or("name = ?", user3.Name).Find(&users).Count(&count).Error; err != nil {
		t.Errorf(fmt.Sprintf("Count should work, but got err %v", err))
	}

	if count != int64(len(users)) {
		t.Errorf("Count() method should get correct value, expect: %v, got %v", count, len(users))
	}

	if err := DB.Model(&User{}).Where("name = ?", user1.Name).Or("name = ?", user3.Name).Count(&count).Find(&users).Error; err != nil {
		t.Errorf(fmt.Sprintf("Count should work, but got err %v", err))
	}

	if count != int64(len(users)) {
		t.Errorf("Count() method should get correct value, expect: %v, got %v", count, len(users))
	}

	DB.Model(&User{}).Where("name = ?", user1.Name).Count(&count1).Or("name in ?", []string{user2.Name, user3.Name}).Count(&count2)
	if count1 != 1 || count2 != 3 {
		t.Errorf("multiple count in chain should works")
	}

	tx := DB.Model(&User{}).Where("name = ?", user1.Name).Session(&gorm.Session{WithConditions: true})
	tx.Count(&count1)
	tx.Or("name in ?", []string{user2.Name, user3.Name}).Count(&count2)
	if count1 != 1 || count2 != 3 {
		t.Errorf("count after new session should works")
	}

	var count3 int64
	if err := DB.Model(&User{}).Where("name in ?", []string{user2.Name, user2.Name, user3.Name}).Group("id").Count(&count3).Error; err != nil {
		t.Errorf("Error happened when count with group, but got %v", err)
	}

	if count3 != 2 {
		t.Errorf("Should get correct count for count with group, but got %v", count3)
	}
}
