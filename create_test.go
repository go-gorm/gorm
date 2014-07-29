package gorm_test

import (
	"reflect"
	"testing"
	"time"
)

func TestCreate(t *testing.T) {
	user := User{Name: "1", Age: 18, Birthday: time.Now(), UserNum: Num(111), PasswordHash: []byte{'f', 'a', 'k', '4'}}

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
}
