package tests_test

import (
	"strconv"
	"testing"

	. "github.com/jinzhu/gorm/tests"
)

func TestNestedPreload(t *testing.T) {
	var user = *GetUser("nested_preload", Config{Pets: 2})

	for idx, pet := range user.Pets {
		pet.Toy = Toy{Name: "toy_nested_preload_" + strconv.Itoa(idx+1)}
	}

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	var user2 User
	DB.Preload("Pets.Toy").Find(&user2, "id = ?", user.ID)

	CheckUser(t, user2, user)
}

func TestNestedPreloadForSlice(t *testing.T) {
	var users = []User{
		*GetUser("slice_nested_preload_1", Config{Pets: 2}),
		*GetUser("slice_nested_preload_2", Config{Pets: 0}),
		*GetUser("slice_nested_preload_3", Config{Pets: 3}),
	}

	for _, user := range users {
		for idx, pet := range user.Pets {
			pet.Toy = Toy{Name: user.Name + "_toy_nested_preload_" + strconv.Itoa(idx+1)}
		}
	}

	if err := DB.Create(&users).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	}

	var userIDs []uint
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}

	var users2 []User
	DB.Preload("Pets.Toy").Find(&users2, "id IN ?", userIDs)

	for idx, user := range users2 {
		CheckUser(t, user, users[idx])
	}
}

func TestPreloadWithConds(t *testing.T) {
}
