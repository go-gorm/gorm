package tests_test

import (
	"testing"

	. "github.com/jinzhu/gorm/tests"
)

func TestFindOrInitialize(t *testing.T) {
	var user1, user2, user3, user4, user5, user6 User
	if err := DB.Where(&User{Name: "find or init", Age: 33}).FirstOrInit(&user1).Error; err != nil {
		t.Errorf("no error should happen when FirstOrInit, but got %v", err)
	}
	if user1.Name != "find or init" || user1.ID != 0 || user1.Age != 33 {
		t.Errorf("user should be initialized with search value")
	}

	DB.Where(User{Name: "find or init", Age: 33}).FirstOrInit(&user2)
	if user2.Name != "find or init" || user2.ID != 0 || user2.Age != 33 {
		t.Errorf("user should be initialized with search value")
	}

	DB.FirstOrInit(&user3, map[string]interface{}{"name": "find or init 2"})
	if user3.Name != "find or init 2" || user3.ID != 0 {
		t.Errorf("user should be initialized with inline search value")
	}

	DB.Where(&User{Name: "find or init"}).Attrs(User{Age: 44}).FirstOrInit(&user4)
	if user4.Name != "find or init" || user4.ID != 0 || user4.Age != 44 {
		t.Errorf("user should be initialized with search value and attrs")
	}

	DB.Where(&User{Name: "find or init"}).Assign("age", 44).FirstOrInit(&user4)
	if user4.Name != "find or init" || user4.ID != 0 || user4.Age != 44 {
		t.Errorf("user should be initialized with search value and assign attrs")
	}

	DB.Save(&User{Name: "find or init", Age: 33})
	DB.Where(&User{Name: "find or init"}).Attrs("age", 44).FirstOrInit(&user5)
	if user5.Name != "find or init" || user5.ID == 0 || user5.Age != 33 {
		t.Errorf("user should be found and not initialized by Attrs")
	}

	DB.Where(&User{Name: "find or init", Age: 33}).FirstOrInit(&user6)
	if user6.Name != "find or init" || user6.ID == 0 || user6.Age != 33 {
		t.Errorf("user should be found with FirstOrInit")
	}

	DB.Where(&User{Name: "find or init"}).Assign(User{Age: 44}).FirstOrInit(&user6)
	if user6.Name != "find or init" || user6.ID == 0 || user6.Age != 44 {
		t.Errorf("user should be found and updated with assigned attrs")
	}
}

func TestFindOrCreate(t *testing.T) {
}
