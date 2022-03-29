package tests_test

import (
	"errors"
	"gorm.io/gorm"
	. "gorm.io/gorm/utils/tests"
	"testing"
)

func TestInvalidParamTypeStruct(t *testing.T) {
	user := User{Name: "TestInvalidParam"}
	DB.Create(&user)

	// panic when update values api
	invalidUser := User{Name: "TestInvalidParam_invalid"}
	invalidUsers := [1]User{invalidUser}
	assertInvalidValueError(t, DB.Create(invalidUser))
	assertInvalidValueError(t, DB.CreateInBatches(invalidUser, 1))
	assertInvalidValueError(t, DB.CreateInBatches(invalidUsers, 1))
	assertInvalidValueError(t, DB.Save(invalidUser))

	// panic when found and update values api
	var invalidQueryUser User
	invalidQueryUser.ID = user.ID
	invalidQueryUsers := [1]User{invalidQueryUser}
	assertInvalidValueError(t, DB.First(invalidQueryUser))
	assertInvalidValueError(t, DB.Take(invalidQueryUser))
	assertInvalidValueError(t, DB.Last(invalidQueryUser))
	assertInvalidValueError(t, DB.Find(invalidQueryUsers))
	assertInvalidValueError(t, DB.FindInBatches(invalidQueryUsers, 1, func(tx *gorm.DB, batch int) error {
		return nil
	}))
	assertInvalidValueError(t, DB.FirstOrInit(invalidQueryUser))
	assertInvalidValueError(t, DB.FirstOrCreate(invalidQueryUser))
	assertInvalidValueError(t, DB.Model(User{}).Scan(invalidQueryUser))
}

func assertInvalidValueError(t *testing.T, tx *gorm.DB) {
	if !errors.Is(tx.Error, gorm.ErrInvalidValue) {
		t.Errorf("should returns error invalid")
	}
}
