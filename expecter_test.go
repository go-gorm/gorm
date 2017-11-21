package gorm_test

import (
	"fmt"
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
	defer db.Close()

	if err != nil {
		t.Fatal(err)
	}
}

func TestQuery(t *testing.T) {
	db, expect, err := gorm.NewDefaultExpecter()

	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("Got here")
	expect.First(&User{})
	db.First(&User{})

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

	in := User{Id: 1}
	out := User{Id: 1, Name: "jinzhu"}

	expect.First(&in).Returns(out)

	db.First(&in)

	if e := expect.AssertExpectations(); e != nil {
		t.Error(e)
	}

	if in.Name != "jinzhu" {
		t.Errorf("Expected %s, got %s", out.Name, in.Name)
	}

	if ne := reflect.DeepEqual(in, out); !ne {
		t.Errorf("Not equal")
	}
}

func TestFindStructDest(t *testing.T) {
	db, expect, err := gorm.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	in := &User{Id: 1}

	expect.Find(in)
	db.Find(&User{Id: 1})

	if e := expect.AssertExpectations(); e != nil {
		t.Error(e)
	}
}

func TestFindSlice(t *testing.T) {
	db, expect, err := gorm.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	in := []User{}
	out := []User{User{Id: 1, Name: "jinzhu"}, User{Id: 2, Name: "itwx"}}

	expect.Find(&in).Returns(&out)
	db.Find(&in)

	if e := expect.AssertExpectations(); e != nil {
		t.Error(e)
	}

	if ne := reflect.DeepEqual(in, out); !ne {
		t.Error("Expected equal slices")
	}
}

func TestMockPreload(t *testing.T) {
	db, expect, err := gorm.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	in := User{Id: 1}
	outEmails := []Email{Email{Id: 1, UserId: 1}, Email{Id: 2, UserId: 1}}
	out := User{Id: 1, Emails: outEmails}

	expect.Preload("Emails").Find(&in).Returns(out)
	db.Preload("Emails").Find(&in)

	// if err := expect.AssertExpectations(); err != nil {
	// 	t.Error(err)
	// }

	if !reflect.DeepEqual(in, out) {
		t.Error("In and out are not equal")
	}
}
