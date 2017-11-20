package gorm_test

import (
	"reflect"
	"testing"

	"github.com/jinzhu/gorm"
)

func TestNewDefaultExpecter(t *testing.T) {
	db, _, err := gorm.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}
}

func TestNewCustomExpecter(t *testing.T) {
	db, _, err := gorm.NewExpecter(gorm.NewSqlmockAdapter, "sqlmock", "mock_gorm_dsn")
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}
}

func TestQuery(t *testing.T) {
	db, expect, err := gorm.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	expect.First(&User{})
	db.LogMode(true).First(&User{})

	if err := expect.AssertExpectations(); err != nil {
		t.Error(err)
	}
}

func TestQueryReturn(t *testing.T) {
	db, expect, err := gorm.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	in := &User{Id: 1}
	expectedOut := User{Id: 1, Name: "jinzhu"}

	expect.First(in).Returns(User{Id: 1, Name: "jinzhu"})

	db.First(in)

	if e := expect.AssertExpectations(); e != nil {
		t.Error(e)
	}

	if in.Name != "jinzhu" {
		t.Errorf("Expected %s, got %s", expectedOut.Name, in.Name)
	}

	if ne := reflect.DeepEqual(*in, expectedOut); !ne {
		t.Errorf("Not equal")
	}
}
