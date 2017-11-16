package gorm_test

import (
	"testing"

	"github.com/iantanwx/gorm"
)

func TestOpenWithSqlmock(t *testing.T) {
	err, _, _ := gorm.NewTestHelper(&gorm.SqlmockAdapter{})

	if err != nil {
		t.Error(err)
	}
}

func TestQuery(t *testing.T) {
	err, db, helper := gorm.NewTestHelper(&gorm.SqlmockAdapter{})

	if err != nil {
		t.Fatal(err.Error())
	}

	helper.ExpectFirst(&User{}).Return(&User{})
	db.First(&User{})
}
