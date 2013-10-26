package gorm

import (
	"fmt"

	"testing"
)

type User struct {
	Name string
}

func getDB() DB {
	db, _ := Open("postgres", "user=gorm dbname=gorm  sslmode=disable")
	return db
}

func TestSaveAndFirst(t *testing.T) {
	// create table "users" ("name" varchar(255));
	db := getDB()
	u := &User{Name: "jinzhu"}
	db.Save(u)

	user := &User{}
	db.First(user)
	if user.Name != "jinzhu" {
		t.Errorf("User should be saved and fetched correctly")
	}

	users := []User{}
	db.Find(&users)
	for _, user := range users {
		if user.Name != "jinzhu" {
			t.Errorf("User should be saved and fetched correctly")
		}
	}
}

func TestWhere(t *testing.T) {
	db := getDB()
	u := &User{Name: "jinzhu"}
	db.Save(u)

	user := &User{}
	db.Where("Name = ?", "jinzhu").First(user)

	fmt.Println(user)
}
