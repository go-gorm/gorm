package gorm

import (
	"reflect"
	"testing"
)

func TestCloneSearch(t *testing.T) {
	s := new(search)
	s.where("name = ?", "jinzhu").order("name").attrs("name", "jinzhu").selects("name, age")

	s1 := s.clone()
	s1.where("age = ?", 20).order("age").attrs("email", "a@e.org").selects("email")

	if reflect.DeepEqual(s.whereClause, s1.whereClause) {
		t.Errorf("Where should be copied")
	}

	if reflect.DeepEqual(s.orders, s1.orders) {
		t.Errorf("Order should be copied")
	}

	if reflect.DeepEqual(s.initAttrs, s1.initAttrs) {
		t.Errorf("initAttrs should be copied")
	}

	if reflect.DeepEqual(s.selectStr, s1.selectStr) {
		t.Errorf("selectStr should be copied")
	}
}
