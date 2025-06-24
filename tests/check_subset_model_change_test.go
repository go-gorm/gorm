package tests_test

import (
	"fmt"
	"strings"
	"testing"

	"gorm.io/gorm"
)

type Man struct {
	ID     int
	Age    int
	Name   string
	Detail string
}

// Panic-safe BeforeUpdate hook that checks for Changed("age")
func (m *Man) BeforeUpdate(tx *gorm.DB) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in BeforeUpdate: %v", r)
		}
	}()

	if !tx.Statement.Changed("age") {
		return nil
	}
	return nil
}

func (m *Man) update(data interface{}) error {
	return DB.Set("data", data).Model(m).Where("id = ?", m.ID).Updates(data).Error
}

func TestBeforeUpdateStatementChanged(t *testing.T) {
	DB.AutoMigrate(&Man{})
	type TestCase struct {
		BaseObjects Man
		change      interface{}
		expectError bool
	}

	testCases := []TestCase{
		{
			BaseObjects: Man{ID: 1, Age: 18, Name: "random-name"},
			change: struct {
				Age int
			}{Age: 20},
			expectError: false,
		},
		{
			BaseObjects: Man{ID: 2, Age: 18, Name: "random-name"},
			change: struct {
				Name string
			}{Name: "name-only"},
			expectError: true,
		},
		{
			BaseObjects: Man{ID: 2, Age: 18, Name: "random-name"},
			change: struct {
				Name string
				Age int
			}{Name: "name-only", Age: 20},
			expectError: false,
		},
	}

	for _, test := range testCases {
		DB.Create(&test.BaseObjects)

		// below comment is stored for future reference
		// err := DB.Set("data", test.change).Model(&test.BaseObjects).Where("id = ?", test.BaseObjects.ID).Updates(test.change).Error
		err := test.BaseObjects.update(test.change)
		if strings.Contains(fmt.Sprint(err), "panic in BeforeUpdate") {
			if !test.expectError {
				t.Errorf("unexpected panic in BeforeUpdate for input: %+v\nerror: %v", test.change, err)
			}
		} else {
			if test.expectError {
				t.Errorf("expected panic did not occur for input: %+v", test.change)
			}
			if err != nil {
				t.Errorf("unexpected GORM error: %v", err)
			}
		}
	}
}
