package gorm_test

import (
	"encoding/hex"
	"math/rand"
	"strings"
	"testing"

	"github.com/jinzhu/gorm"
	"reflect"
)

func NameIn1And2(d *gorm.DB) *gorm.DB {
	return d.Where("name in (?)", []string{"ScopeUser1", "ScopeUser2"})
}

func NameIn2And3(d *gorm.DB) *gorm.DB {
	return d.Where("name in (?)", []string{"ScopeUser2", "ScopeUser3"})
}

func NameIn(names []string) func(d *gorm.DB) *gorm.DB {
	return func(d *gorm.DB) *gorm.DB {
		return d.Where("name in (?)", names)
	}
}

func TestScopes(t *testing.T) {
	user1 := User{Name: "ScopeUser1", Age: 1}
	user2 := User{Name: "ScopeUser2", Age: 1}
	user3 := User{Name: "ScopeUser3", Age: 2}
	DB.Save(&user1).Save(&user2).Save(&user3)

	var users1, users2, users3 []User
	DB.Scopes(NameIn1And2).Find(&users1)
	if len(users1) != 2 {
		t.Errorf("Should found two users's name in 1, 2")
	}

	DB.Scopes(NameIn1And2, NameIn2And3).Find(&users2)
	if len(users2) != 1 {
		t.Errorf("Should found one user's name is 2")
	}

	DB.Scopes(NameIn([]string{user1.Name, user3.Name})).Find(&users3)
	if len(users3) != 2 {
		t.Errorf("Should found two users's name in 1, 3")
	}
}

func randName() string {
	data := make([]byte, 8)
	rand.Read(data)

	return "n-" + hex.EncodeToString(data)
}

func TestValuer(t *testing.T) {
	name := randName()

	origUser := User{Name: name, Age: 1, Password: EncryptedData("pass1"), PasswordHash: []byte("abc")}
	if err := DB.Save(&origUser).Error; err != nil {
		t.Errorf("No error should happen when saving user, but got %v", err)
	}

	var user2 User
	if err := DB.Where("name = ? AND password = ? AND password_hash = ?", name, EncryptedData("pass1"), []byte("abc")).First(&user2).Error; err != nil {
		t.Errorf("No error should happen when querying user with valuer, but got %v", err)
	}
}

func TestFailedValuer(t *testing.T) {
	name := randName()

	err := DB.Exec("INSERT INTO users(name, password) VALUES(?, ?)", name, EncryptedData("xpass1")).Error

	if err == nil {
		t.Errorf("There should be an error should happen when insert data")
	} else if !strings.HasPrefix(err.Error(), "Should not start with") {
		t.Errorf("The error should be returned from Valuer, but get %v", err)
	}
}

func TestAfterFieldScanCallback(t *testing.T) {
	model := WithFieldAfterScanCallback{}
	model.Name1 = &AfterScanFieldPtr{data: randName()}
	model.Name2 = AfterScanFieldPtr{data: randName()}
	model.Name3 = &AfterScanField{data: randName()}
	model.Name4 = AfterScanField{data: randName()}

	if err := DB.Save(&model).Error; err != nil {
		t.Errorf("No error should happen when saving WithFieldAfterScanCallback, but got %v", err)
	}

	var model2 WithFieldAfterScanCallback
	if err := DB.Where("id = ?", model.ID).First(&model2).Error; err != nil {
		t.Errorf("No error should happen when querying WithFieldAfterScanCallback with valuer, but got %v", err)
	}

	dotest := func(i int, value string, field AfterScanFieldInterface) {
		if field.CalledFieldIsNill() {
			t.Errorf("Expected Name%v.calledField, but got nil", i)
		}

		if field.CalledScopeIsNill() {
			t.Errorf("Expected Name%v.calledScope, but got nil", i)
		}

		if field.Data() != value {
			t.Errorf("Expected Name%v.data %q, but got %q", i, value, field.Data())
		}
	}

	dotest(1, model.Name1.data, model2.Name1)
	dotest(2, model.Name2.data, &model2.Name2)
	dotest(3, model.Name3.data, model2.Name3)
	dotest(4, model.Name4.data, &model2.Name4)
}

func TestAfterFieldScanDisableCallback(t *testing.T) {
	model := WithFieldAfterScanCallback{}
	model.Name1 = &AfterScanFieldPtr{data: randName()}

	if err := DB.Save(&model).Error; err != nil {
		t.Errorf("No error should happen when saving WithFieldAfterScanCallback, but got %v", err)
	}

	run := func(key string) {
		DB := DB.Set(key, true)
		var model2 WithFieldAfterScanCallback
		if err := DB.Where("id = ?", model.ID).First(&model2).Error; err != nil {
			t.Errorf("%q: No error should happen when querying WithFieldAfterScanCallback with valuer, but got %v", key, err)
		}

		dotest := func(i int, value string, field AfterScanFieldInterface) {
			if !field.CalledFieldIsNill() {
				t.Errorf("%q: Expected Name%v.calledField is not nil", key, i)
			}

			if !field.CalledScopeIsNill() {
				t.Errorf("%q: Expected Name%v.calledScope is not nil", key, i)
			}
		}

		dotest(1, model.Name1.data, model2.Name1)
	}

	run("gorm:disable_after_scan")
	typ := reflect.ValueOf(model).Type()
	run("gorm:disable_after_scan:" + typ.PkgPath() + "." + typ.Name())
}

func TestAfterFieldScanInvalidCallback(t *testing.T) {
	model := WithFieldAfterScanInvalidCallback{}
	model.Name = InvalidAfterScanField{AfterScanField{data: randName()}}

	if err := DB.Save(&model).Error; err != nil {
		t.Errorf("No error should happen when saving WithFieldAfterScanCallback, but got %v", err)
	}

	var model2 WithFieldAfterScanInvalidCallback
	if err := DB.Where("id = ?", model.ID).First(&model2).Error; err != nil {
		if !strings.Contains(err.Error(), "Invalid AfterScan method callback") {
			t.Errorf("No error should happen when querying WithFieldAfterScanCallback with valuer, but got %v", err)
		}
	} else {
		t.Errorf("Expected error, but got nil")
	}
}
