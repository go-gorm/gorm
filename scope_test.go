package gorm_test

import (
	"encoding/hex"
	"math/rand"
	"strings"
	"testing"

	"github.com/jinzhu/gorm"
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
