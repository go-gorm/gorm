package gorm_test

import "testing"

func TestFirstAndLast(t *testing.T) {
	db.Save(&User{Name: "user1", Emails: []Email{{Email: "user1@example.com"}}})
	db.Save(&User{Name: "user2", Emails: []Email{{Email: "user2@example.com"}}})

	var user1, user2, user3, user4 User
	db.First(&user1)
	db.Order("id").Find(&user2)

	db.Last(&user3)
	db.Order("id desc").Find(&user4)
	if user1.Id != user2.Id || user3.Id != user4.Id {
		t.Errorf("First and Last should by order by primary key")
	}

	var users []User
	db.First(&users)
	if len(users) != 1 {
		t.Errorf("Find first record as slice")
	}

	if db.Joins("left join emails on emails.user_id = users.id").First(&User{}).Error != nil {
		t.Errorf("Should not raise any error when order with Join table")
	}
}

func TestFirstAndLastWithNoStdPrimaryKey(t *testing.T) {
	db.Save(&Animal{Name: "animal1"})
	db.Save(&Animal{Name: "animal2"})

	var animal1, animal2, animal3, animal4 Animal
	db.First(&animal1)
	db.Order("counter").Find(&animal2)

	db.Last(&animal3)
	db.Order("counter desc").Find(&animal4)
	if animal1.Counter != animal2.Counter || animal3.Counter != animal4.Counter {
		t.Errorf("First and Last should work correctly")
	}
}

func TestFindAsSliceOfPointers(t *testing.T) {
	db.Save(&User{Name: "user"})

	var users []User
	db.Find(&users)

	var userPointers []*User
	db.Find(&userPointers)

	if len(users) == 0 || len(users) != len(userPointers) {
		t.Errorf("Find slice of pointers")
	}
}
