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

	if reflect.DeepEqual(s.WhereConditions, s1.WhereConditions) {
		t.Errorf("Where should be copied")
	}

	if reflect.DeepEqual(s.Orders, s1.Orders) {
		t.Errorf("Order should be copied")
	}

	if reflect.DeepEqual(s.InitAttrs, s1.InitAttrs) {
		t.Errorf("InitAttrs should be copied")
	}

	if reflect.DeepEqual(s.Select, s1.Select) {
		t.Errorf("selectStr should be copied")
	}
}
