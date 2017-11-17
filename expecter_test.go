package gorm_test

import (
	"testing"

	"github.com/jinzhu/gorm"
)

func TestNewDefaultExpecter(t *testing.T) {
	err, db, _ := gorm.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}
}

func TestNewCustomExpecter(t *testing.T) {
	err, db, _ := gorm.NewExpecter(gorm.NewSqlmockAdapter, "sqlmock", "mock_gorm_dsn")
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}
}

func TestQuery(t *testing.T) {
	err, db, expect := gorm.NewDefaultExpecter()
	defer func() {
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}

	expect.ExpectFirst(&User{}).Returns(&User{})
	db.First(&User{})
}
