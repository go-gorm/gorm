package tests_test

import (
	"testing"
	"time"

	"github.com/jinzhu/gorm"
	. "github.com/jinzhu/gorm/tests"
)

func TestUpdate(t *testing.T) {
	var (
		users = []*User{
			GetUser("update-1", Config{}),
			GetUser("update-2", Config{}),
			GetUser("update-3", Config{}),
		}
		user          = users[1]
		lastUpdatedAt time.Time
	)

	checkUpdatedAtChanged := func(name string, n time.Time) {
		if n.UnixNano() == lastUpdatedAt.UnixNano() {
			t.Errorf("%v: user's updated at should be changed, but got %v, was %v", name, n, lastUpdatedAt)
		}
		lastUpdatedAt = n
	}

	checkOtherData := func(name string) {
		var first, last User
		if err := DB.Where("id = ?", users[0].ID).First(&first).Error; err != nil {
			t.Errorf("errors happened when query before user: %v", err)
		}
		CheckUser(t, first, *users[0])

		if err := DB.Where("id = ?", users[2].ID).First(&last).Error; err != nil {
			t.Errorf("errors happened when query after user: %v", err)
		}
		CheckUser(t, last, *users[2])
	}

	if err := DB.Create(&users).Error; err != nil {
		t.Fatalf("errors happened when create: %v", err)
	} else if user.ID == 0 {
		t.Fatalf("user's primary value should not zero, %v", user.ID)
	} else if user.UpdatedAt.IsZero() {
		t.Fatalf("user's updated at should not zero, %v", user.UpdatedAt)
	}
	lastUpdatedAt = user.UpdatedAt

	if err := DB.Model(user).Update("Age", 10).Error; err != nil {
		t.Errorf("errors happened when update: %v", err)
	} else if user.Age != 10 {
		t.Errorf("Age should equals to 10, but got %v", user.Age)
	}
	checkUpdatedAtChanged("Update", user.UpdatedAt)
	checkOtherData("Update")

	var result User
	if err := DB.Where("id = ?", user.ID).First(&result).Error; err != nil {
		t.Errorf("errors happened when query: %v", err)
	} else {
		CheckUser(t, result, *user)
	}

	values := map[string]interface{}{"Active": true, "age": 5}
	if err := DB.Model(user).Updates(values).Error; err != nil {
		t.Errorf("errors happened when update: %v", err)
	} else if user.Age != 5 {
		t.Errorf("Age should equals to 5, but got %v", user.Age)
	} else if user.Active != true {
		t.Errorf("Active should be true, but got %v", user.Active)
	}
	checkUpdatedAtChanged("Updates with map", user.UpdatedAt)
	checkOtherData("Updates with map")

	var result2 User
	if err := DB.Where("id = ?", user.ID).First(&result2).Error; err != nil {
		t.Errorf("errors happened when query: %v", err)
	} else {
		CheckUser(t, result2, *user)
	}

	if err := DB.Model(user).Updates(User{Age: 2}).Error; err != nil {
		t.Errorf("errors happened when update: %v", err)
	} else if user.Age != 2 {
		t.Errorf("Age should equals to 2, but got %v", user.Age)
	}
	checkUpdatedAtChanged("Updates with struct", user.UpdatedAt)
	checkOtherData("Updates with struct")

	var result3 User
	if err := DB.Where("id = ?", user.ID).First(&result3).Error; err != nil {
		t.Errorf("errors happened when query: %v", err)
	} else {
		CheckUser(t, result3, *user)
	}

	user.Active = false
	user.Age = 1
	if err := DB.Save(user).Error; err != nil {
		t.Errorf("errors happened when update: %v", err)
	} else if user.Age != 1 {
		t.Errorf("Age should equals to 1, but got %v", user.Age)
	} else if user.Active != false {
		t.Errorf("Active should equals to false, but got %v", user.Active)
	}
	checkUpdatedAtChanged("Save", user.UpdatedAt)
	checkOtherData("Save")

	var result4 User
	if err := DB.Where("id = ?", user.ID).First(&result4).Error; err != nil {
		t.Errorf("errors happened when query: %v", err)
	} else {
		CheckUser(t, result4, *user)
	}
}

func TestUpdates(t *testing.T) {
	var users = []*User{
		GetUser("updates_01", Config{}),
		GetUser("updates_02", Config{}),
	}

	DB.Create(&users)
	lastUpdatedAt := users[0].UpdatedAt

	// update with map
	DB.Model(users[0]).Updates(map[string]interface{}{"name": "updates_01_newname", "age": 100})
	if users[0].Name != "updates_01_newname" || users[0].Age != 100 {
		t.Errorf("Record should be updated also with map")
	}

	if users[0].UpdatedAt.UnixNano() == lastUpdatedAt.UnixNano() {
		t.Errorf("User's updated at should be changed, but got %v, was %v", users[0].UpdatedAt.UnixNano(), lastUpdatedAt)
	}

	// user2 should not be updated
	var user1, user2 User
	DB.First(&user1, users[0].ID)
	DB.First(&user2, users[1].ID)
	CheckUser(t, user1, *users[0])
	CheckUser(t, user2, *users[1])

	// update with struct
	DB.Table("users").Where("name in ?", []string{users[1].Name}).Updates(User{Name: "updates_02_newname"})

	var user3 User
	if DB.First(&user3, "name = ?", "updates_02_newname").RecordNotFound() {
		t.Errorf("User2's name should be updated")
	}
	AssertEqual(t, user2.UpdatedAt, user3.UpdatedAt)

	// update with gorm exprs
	DB.Model(&user3).Updates(map[string]interface{}{"age": gorm.Expr("age + ?", 100)})
	var user4 User
	DB.First(&user4, user3.ID)

	user3.Age += 100
	AssertEqual(t, user4.UpdatedAt, user3.UpdatedAt)
}
