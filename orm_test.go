package gorm

import (
	"fmt"

	"testing"
)

type User struct {
	Name string
}

func TestWhere(t *testing.T) {
	db, err := Open("postgres", "user=gorm dbname=gorm")

	if err != err {
		t.Errorf("Error should be nil")
	}
	orm := db.Where("id = $1", 1, 3, 4, []int64{1, 2, 3}).Where("name = $1", "jinzhu")

	user := &User{}
	orm.First(user)
	fmt.Println(user)
}
