package tests_test

import (
	"context"
	"testing"

	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
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
	var users = []*User{
		GetUser("ScopeUser1", Config{}),
		GetUser("ScopeUser2", Config{}),
		GetUser("ScopeUser3", Config{}),
	}

	DB.Create(&users)

	var users1, users2, users3 []User
	DB.Scopes(NameIn1And2).Find(&users1)
	if len(users1) != 2 {
		t.Errorf("Should found two users's name in 1, 2, but got %v", len(users1))
	}

	DB.Scopes(NameIn1And2, NameIn2And3).Find(&users2)
	if len(users2) != 1 {
		t.Errorf("Should found one user's name is 2, but got %v", len(users2))
	}

	DB.Scopes(NameIn([]string{users[0].Name, users[2].Name})).Find(&users3)
	if len(users3) != 2 {
		t.Errorf("Should found two users's name in 1, 3, but got %v", len(users3))
	}

	db := DB.Scopes(func(tx *gorm.DB) *gorm.DB {
		return tx.Table("custom_table")
	}).Session(&gorm.Session{})

	db.AutoMigrate(&User{})
	if db.Find(&User{}).Statement.Table != "custom_table" {
		t.Errorf("failed to call Scopes")
	}

	result := DB.Scopes(NameIn1And2, func(tx *gorm.DB) *gorm.DB {
		return tx.Session(&gorm.Session{})
	}).Find(&users1)

	if result.RowsAffected != 2 {
		t.Errorf("Should found two users's name in 1, 2, but got %v", result.RowsAffected)
	}

	var maxId int64
	userTable := func(db *gorm.DB) *gorm.DB {
		return db.WithContext(context.Background()).Table("users")
	}
	if err := DB.Scopes(userTable).Select("max(id)").Scan(&maxId).Error; err != nil {
		t.Errorf("select max(id)")
	}
}
