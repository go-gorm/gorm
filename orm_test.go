package gorm

import (
	"fmt"
	"time"

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
	db := getDB()
	u := &User{Name: "jinzhu"}
	fmt.Println("*******")
	fmt.Println(db.Save(u).Sql)
	fmt.Println(db.Save(u).Error)
	fmt.Println(time.Now().String())

	user := &User{}
	db.First(&user)
	if user.Name != "jinzhu" {
		t.Errorf("User should be saved and fetched correctly")
	}
}

func TestWhere(t *testing.T) {
	db := getDB()
	orm := db.Where("id = $1", 1, 3, 4, []int64{1, 2, 3}).Where("name = $1", "jinzhu")
	user := &User{}
	orm.First(user)
	fmt.Println(user)
}
