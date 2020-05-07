package gorm

import (
	"fmt"
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

func TestWhereCloneCorruption(t *testing.T) {
	for whereCount := 1; whereCount <= 8; whereCount++ {
		t.Run(fmt.Sprintf("w=%d", whereCount), func(t *testing.T) {
			s := new(search)
			for w := 0; w < whereCount; w++ {
				s = s.clone().Where(fmt.Sprintf("w%d = ?", w), fmt.Sprintf("value%d", w))
			}
			if len(s.whereConditions) != whereCount {
				t.Errorf("s: where count should be %d", whereCount)
			}

			q1 := s.clone().Where("finalThing = ?", "THING1")
			q2 := s.clone().Where("finalThing = ?", "THING2")

			if reflect.DeepEqual(q1.whereConditions, q2.whereConditions) {
				t.Errorf("Where conditions should be different")
			}
		})
	}
}
