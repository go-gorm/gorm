package gorm

import (
	"reflect"
	"testing"
)

func TestCloneSearch(t *testing.T) {
	s := new(search)
	s.Where("name = ?", "jinzhu").Order("name").Attrs("name", "jinzhu").Select("name, age")

	s1 := s.clone()
	s1.Where("age = ?", 20).Order("age").Attrs("email", "a@e.org").Select("email")

	if reflect.DeepEqual(s.whereConditions, s1.whereConditions) {
		t.Errorf("Where should be copied")
	}

	if reflect.DeepEqual(s.orders, s1.orders) {
		t.Errorf("Order should be copied")
	}

	if reflect.DeepEqual(s.initAttrs, s1.initAttrs) {
		t.Errorf("InitAttrs should be copied")
	}

	if reflect.DeepEqual(s.Select, s1.Select) {
		t.Errorf("selectStr should be copied")
	}
}
