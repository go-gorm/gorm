package gorm_test

import (
	"testing"

	"github.com/jinzhu/gorm"
)

var helper *gorm.TestHelper
var mockDb *gorm.DB

func TestOpenWithSqlmock(t *testing.T) {
	err, db, h := gorm.NewTestHelper(&gorm.SqlmockAdapter{})

	if err != nil {
		t.Fatal(err)
	}

	helper = h
	mockDb = db
}

func TestQuery(t *testing.T) {
	helper.ExpectFirst(&User{}).Return(&User{})
	mockDb.First(&User{})
}
