package tests_test

import (
	"reflect"
	"strconv"
	"testing"

	. "github.com/jinzhu/gorm/tests"
)

func TestFind(t *testing.T) {
	var users = []User{
		*GetUser("find", Config{}),
		*GetUser("find", Config{}),
		*GetUser("find", Config{}),
	}

	if err := DB.Create(&users).Error; err != nil {
		t.Fatalf("errors happened when create users: %v", err)
	}

	t.Run("First", func(t *testing.T) {
		var first User
		if err := DB.Where("name = ?", "find").First(&first).Error; err != nil {
			t.Errorf("errors happened when query first: %v", err)
		} else {
			CheckUser(t, first, users[0])
		}
	})

	t.Run("Last", func(t *testing.T) {
		var last User
		if err := DB.Where("name = ?", "find").Last(&last).Error; err != nil {
			t.Errorf("errors happened when query last: %v", err)
		} else {
			CheckUser(t, last, users[2])
		}
	})

	var all []User
	if err := DB.Where("name = ?", "find").Find(&all).Error; err != nil || len(all) != 3 {
		t.Errorf("errors happened when query find: %v, length: %v", err, len(all))
	} else {
		for idx, user := range users {
			t.Run("FindAll#"+strconv.Itoa(idx+1), func(t *testing.T) {
				CheckUser(t, all[idx], user)
			})
		}
	}

	t.Run("FirstMap", func(t *testing.T) {
		var first = map[string]interface{}{}
		if err := DB.Model(&User{}).Where("name = ?", "find").First(first).Error; err != nil {
			t.Errorf("errors happened when query first: %v", err)
		} else {
			for _, name := range []string{"Name", "Age", "Birthday"} {
				t.Run(name, func(t *testing.T) {
					dbName := DB.NamingStrategy.ColumnName("", name)
					reflectValue := reflect.Indirect(reflect.ValueOf(users[0]))
					AssertEqual(t, first[dbName], reflectValue.FieldByName(name).Interface())
				})
			}
		}
	})

	var allMap = []map[string]interface{}{}
	if err := DB.Model(&User{}).Where("name = ?", "find").Find(&allMap).Error; err != nil {
		t.Errorf("errors happened when query first: %v", err)
	} else {
		for idx, user := range users {
			t.Run("FindAllMap#"+strconv.Itoa(idx+1), func(t *testing.T) {
				for _, name := range []string{"Name", "Age", "Birthday"} {
					t.Run(name, func(t *testing.T) {
						dbName := DB.NamingStrategy.ColumnName("", name)
						reflectValue := reflect.Indirect(reflect.ValueOf(user))
						AssertEqual(t, allMap[idx][dbName], reflectValue.FieldByName(name).Interface())
					})
				}
			})
		}
	}
}
