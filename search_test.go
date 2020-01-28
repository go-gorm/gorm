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

func TestPage(t *testing.T) {
	s := new(search)
	s.Where("name = ?", "jinzhu").Order("id").Page(1).Per(17)
	if s.page != 1 {
		t.Error("page should be 1")
	}
	if s.perPage != 17 {
		t.Error("perPage should be 17")
	}
	if s.limit != 17 {
		t.Error("limit should be 17")
	}
	if s.offset != 0 {
		t.Error("offset should be 0")
	}

	s = new(search)
	s.Where("name = ?", "jinzhu").Order("id").Per(17).Page(2)
	if s.offset != 17 {
		t.Error("offset should be 17")
	}

	s = new(search).Page(0).Per(0)
	if s.page != 1 || s.perPage != defaultPerPage {
		t.Error("page should be 1, perPage should be default per page")
	}

	s = new(search).Page("-3").Per("0")
	if s.page != 1 || s.perPage != defaultPerPage || s.offset != 0 || s.limit != defaultPerPage {
		t.Error("page should be 1, offset should be 0, perPage, limit should be default per page")
	}

	s = new(search).MaxPerPage(50).Page(2).Per("60")
	if s.page != 2 || s.perPage != 50 || s.offset != 50 || s.limit != 50 {
		t.Error("page should be 1, offset, limit, perPage should be 50")
	}
}
