package tests

import (
	"reflect"
	"testing"

	"github.com/jinzhu/gorm"
)

// RunCreateSuit run create suit
func RunCreateSuit(db *gorm.DB, t *testing.T) {
	testInsert(db, t, FormatWithMsg("Insert"))
	testBatchInsert(db, t, FormatWithMsg("BatchInsert"))
}

func testInsert(db *gorm.DB, t *testing.T, format Format) {
	type User struct {
		gorm.Model
		Name string
		Age  int
	}

	user := User{Name: "name1", Age: 10}

	db.Create(&user)

	var newUser User
	if err := db.Find(&newUser, "id = ?", user.ID).Error; err != nil {
		t.Error(format(err))
	}

	if !reflect.DeepEqual(newUser, user) {
		t.Error(format("User should be equal, but got %#v, should be %#v", newUser, user))
	}
}

func testBatchInsert(db *gorm.DB, t *testing.T, format Format) {
	type User struct {
		gorm.Model
		Name string
		Age  int
	}

	users := []*User{{Name: "name1", Age: 10}, {Name: "name2", Age: 20}, {Name: "name3", Age: 30}}

	db.Create(users)

	for _, user := range users {
		if user.ID == 0 {
			t.Error(format("User should have primary key"))
		}

		var newUser User
		if err := db.Find(&newUser, "id = ?", user.ID).Error; err != nil {
			t.Error(format(err))
		}

		if !reflect.DeepEqual(&newUser, user) {
			t.Errorf(format("User should be equal, but got %#v, should be %#v", newUser, user))
		}
	}
}
