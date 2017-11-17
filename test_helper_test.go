package gorm_test

import (
	"testing"

	"github.com/jinzhu/gorm"
)

func TestHelperOpen(t *testing.T) {
	err, helper, db := gorm.NewTestHelper(&gorm.SqlmockAdapter{})

	defer func() {
		helper.Close()
		db.Close()
	}()

	if err != nil {
		t.Fatal(err)
	}
}

func TestHelperClose(t *testing.T) {
	err, helper, _ := gorm.NewTestHelper(&gorm.SqlmockAdapter{})

	closeErr := helper.Close()

	if err != nil {
		t.Fatal(closeErr)
	}
}

// func TestQuery(t *testing.T) {
// 	helper.ExpectFirst(&User{}).Return(&User{})
// 	mockDb.First(&User{})
// }
