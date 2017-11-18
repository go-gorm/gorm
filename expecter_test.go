package gorm_test

import (
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
	db.First(&User{})

	if err := expect.AssertExpectations(); err != nil {
		t.Error(err)
	}
}
