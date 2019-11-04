package gorm_test

import (
	"encoding/hex"
	"math/rand"
	"os"
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

func TestDropTableWithTableOptions(t *testing.T) {
	type UserWithOptions struct {
		gorm.Model
	}
	DB.AutoMigrate(&UserWithOptions{})

	DB = DB.Set("gorm:table_options", "CHARSET=utf8")
	err := DB.DropTable(&UserWithOptions{}).Error
	if err != nil {
		t.Errorf("Table must be dropped, got error %s", err)
	}
}

func TestBytesValueAsArray(t *testing.T) {
	if dialect := os.Getenv("GORM_DIALECT"); dialect != "postgres" && dialect != "mysql" {
		t.Skip("Skipping this because only postgres and mysql support bytes as string")
	}

	users := []*User{
		&User{Name: "bytesValueAsArray1", Age: 1},
		&User{Name: "bytesValueAsArray2", Age: 2},
	}

	for _, user := range users {
		if err := DB.Save(user).Error; err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		for _, user := range users {
			if err := DB.Delete(user).Error; err != nil {
				t.Fatal(err)
			}
		}
	}()

	var user User
	db := DB.New()
	if err := db.Where("name = ?", []byte(users[0].Name)).First(&user).Error; err != nil {
		t.Error(err)
	}
	if user.Id != users[0].Id {
		t.Errorf("user.Id expected %d, but got %d", users[0].Id, user.Id)
	}

	var ids []int64
	db = DB.New()
	if err := db.Model(&User{}).
		Set("gorm:bytes_value_as_array", true).
		Where("name LIKE ?", "bytesValueAsArray%").
		Where("age IN (?)", []byte{byte(users[0].Age), byte(users[1].Age)}).
		Order("age").
		Pluck("id", &ids).
		Error; err != nil {
		t.Error(err)
	}
	if len(ids) != 2 {
		t.Errorf("ids length expected 2, but got %d", len(ids))
	}
	if ids[0] != users[0].Id {
		t.Errorf("ids[0] expected %d, but got %d", users[0].Id, ids[0])
	}
	if ids[1] != users[1].Id {
		t.Errorf("ids[1] expected %d, but got %d", users[0].Id, ids[0])
	}
}
