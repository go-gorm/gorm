package tests_test

import (
	"sort"
	"testing"

	. "github.com/jinzhu/gorm/tests"
)

func TestJoins(t *testing.T) {
	user := *GetUser("joins-1", Config{Company: true, Manager: true, Account: true})

	DB.Create(&user)

	var user2 User
	if err := DB.Joins("Company").Joins("Manager").Joins("Account").First(&user2, "users.name = ?", user.Name).Error; err != nil {
		t.Fatalf("Failed to load with joins, got error: %v", err)
	}

	CheckUser(t, user2, user)
}

func TestJoinsForSlice(t *testing.T) {
	users := []User{
		*GetUser("slice-joins-1", Config{Company: true, Manager: true, Account: true}),
		*GetUser("slice-joins-2", Config{Company: true, Manager: true, Account: true}),
		*GetUser("slice-joins-3", Config{Company: true, Manager: true, Account: true}),
	}

	DB.Create(&users)

	var userIDs []uint
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}

	var users2 []User
	if err := DB.Joins("Company").Joins("Manager").Joins("Account").Find(&users2, "users.id IN ?", userIDs).Error; err != nil {
		t.Fatalf("Failed to load with joins, got error: %v", err)
	} else if len(users2) != len(users) {
		t.Fatalf("Failed to load join users, got: %v, expect: %v", len(users2), len(users))
	}

	sort.Slice(users2, func(i, j int) bool {
		return users2[i].ID > users2[j].ID
	})

	sort.Slice(users, func(i, j int) bool {
		return users[i].ID > users[j].ID
	})

	for idx, user := range users {
		CheckUser(t, user, users2[idx])
	}
}
