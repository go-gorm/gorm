package gorm

import (
	"fmt"
	"github.com/fwhezfwhez/gorm"
	"testing"

)

func TestDB_DataSource(t *testing.T) {
	source := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=%s password=%s",
		"localhost", "postgres", "test", "disable", "123")
	db, er := Open("postgres", source)
	if er != nil {
		t.Fatal(er.Error())
	}
	fmt.Println(db.DataSource())
}
func TestDB_CopyIn(t *testing.T) {
	source := fmt.Sprintf("host=%s user=%s dbname=%s sslmode=%s password=%s",
		"localhost", "postgres", "test", "disable", "123")
	db, er := gorm.Open("postgres", source)
	if er != nil {
		t.Fatal(er.Error())
	}
	db.Exec("create table if not exists example(name varchar, age integer)")
	var args = make([][]interface{}, 0)
	args = append(args, []interface{}{
		"tom", 9,
	}, []interface{}{
		"sara", 10,
	}, []interface{}{
		"jim", 11,
	})
	e := db.CopyIn(true, "example", args, "name", "age")
	if e != nil {
		t.Fatal(e.Error())
	}
	type Example struct{
		Name string
		Age int
	}
	var examples = make([]Example,0)
	e=db.Raw("select * from example").Find(&examples).Error
	if e!=nil {
		t.Fatal(e.Error())
	}
	fmt.Println(examples)
}
