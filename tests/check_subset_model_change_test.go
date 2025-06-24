package tests_test

import (
	"fmt"
	"testing"
)

type Man struct {
	ID  int
	Age int
	Name string
}

func TestBeforeUpdateStatementChanged(t *testing.T) {
	type TestCase struct {
		BaseObjects Man
		change interface{}
	}
	fmt.Println("Running Eshan Jogwar Test")
	testCases := []TestCase{
		{
			BaseObjects: Man{ID: 12231234, Age: 18, Name: "random-name"},
			change: struct {
				Age int
			}{Age: 20},
		},
		{
			BaseObjects: Man{ID: 12231234, Age: 18, Name: "random-name"},
			change: struct {
				Age int
				Name string
			}{Age: 20, Name: "another-random-name"},
		},
	}

	for _, test := range testCases {
		// err := test.BaseObjects.update(test.change)
		err := DB.Set("data", test.change).Model(test.BaseObjects).Where("id = ?", test.BaseObjects.ID).Updates(test.change).Error
		if err != nil {
			t.Errorf(err.Error())
		}
	}
}