package gorm_test

import (
	"reflect"
	"testing"
	"time"
)

func TestCreate(t *testing.T) {
	float := 35.03554004971999
	user := User{Name: "CreateUser", Age: 18, Birthday: time.Now(), UserNum: Num(111), PasswordHash: []byte{'f', 'a', 'k', '4'}, Latitude: float}

	if !db.NewRecord(user) || !db.NewRecord(&user) {
		t.Error("User should be new record before create")
	}

	if count := db.Save(&user).RowsAffected; count != 1 {
		t.Error("There should be one record be affected when create record")
	}

	if db.NewRecord(user) || db.NewRecord(&user) {
		t.Error("User should not new record after save")
	}

	var newUser User
	db.First(&newUser, user.Id)

	if !reflect.DeepEqual(newUser.PasswordHash, []byte{'f', 'a', 'k', '4'}) {
		t.Errorf("User's PasswordHash should be saved ([]byte)")
	}

	if newUser.Age != 18 {
		t.Errorf("User's Age should be saved (int)")
	}

	if newUser.UserNum != Num(111) {
		t.Errorf("User's UserNum should be saved (custom type)")
	}

	if newUser.Latitude != float {
		t.Errorf("Float64 should not be changed after save")
	}

	if user.CreatedAt.IsZero() {
		t.Errorf("Should have created_at after create")
	}

	if newUser.CreatedAt.IsZero() {
		t.Errorf("Should have created_at after create")
	}

	db.Model(user).Update("name", "create_user_new_name")
	db.First(&user, user.Id)
	if user.CreatedAt != newUser.CreatedAt {
		t.Errorf("CreatedAt should not be changed after update")
	}
}

func TestCreateWithNoStdPrimaryKey(t *testing.T) {
	animal := Animal{Name: "Ferdinand"}
	if db.Save(&animal).Error != nil {
		t.Errorf("No error should happen when create an record without std primary key")
	}

	if animal.Counter == 0 {
		t.Errorf("No std primary key should be filled value after create")
	}
}

func TestAnonymousScanner(t *testing.T) {
	user := User{Name: "anonymous_scanner", Role: Role{Name: "admin"}}
	db.Save(&user)

	var user2 User
	db.First(&user2, "name = ?", "anonymous_scanner")
	if user2.Role.Name != "admin" {
		t.Errorf("Should be able to get anonymous scanner")
	}

	if !user2.IsAdmin() {
		t.Errorf("Should be able to get anonymous scanner")
	}
}

func TestAnonymousField(t *testing.T) {
	user := User{Name: "anonymous_field", Company: Company{Name: "company"}}
	db.Save(&user)

	var user2 User
	db.First(&user2, "name = ?", "anonymous_field")
	db.Model(&user2).Related(&user2.Company)
	if user2.Company.Name != "company" {
		t.Errorf("Should be able to get anonymous field")
	}
}
