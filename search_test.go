package gorm

import (
	"reflect"
	"testing"
)

func TestCloneSearch(t *testing.T) {
	s := new(search)
	s.Where("name = ?", "jinzhu").Order("name").Attrs("name", "jinzhu").Selects("name, age")

	s1 := s.clone()
	s1.Where("age = ?", 20).Order("age").Attrs("email", "a@e.org").Selects("email")

	if reflect.DeepEqual(s.whereConditions, s1.whereConditions) {
		t.Errorf("Where should be copied")
	}

	if reflect.DeepEqual(s.orders, s1.orders) {
		t.Errorf("Order should be copied")
	}

	if reflect.DeepEqual(s.initAttrs, s1.initAttrs) {
		t.Errorf("InitAttrs should be copied")
	}

	if reflect.DeepEqual(s.Selects, s1.Selects) {
		t.Errorf("selectStr should be copied")
	}
}
