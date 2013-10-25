package gorm

import (
	"fmt"

	"testing"
)

func TestWhere(t *testing.T) {
	db, err := Open("postgres", "user=gorm dbname=gorm")

	if err != err {
		t.Errorf("Error should be nil")
	}
	orm := db.Where("id = $1", 1).Where("name = $1", "jinzhu")
	fmt.Println(orm.whereClause)
}
