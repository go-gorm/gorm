package utils

import (
	"strings"
	"testing"
)

func TestIsValidDBNameChar(t *testing.T) {
	for _, db := range []string{"db", "dbName", "db_name", "db1", "1dbname", "db$name"} {
		if fields := strings.FieldsFunc(db, IsValidDBNameChar); len(fields) != 1 {
			t.Fatalf("failed to parse db name %v", db)
		}
	}
}

func TestToStringKey(t *testing.T) {
	cases := []struct {
		values []interface{}
		key    string
	}{
		{[]interface{}{"a"}, "a"},
		{[]interface{}{1, 2, 3}, "1_2_3"},
		{[]interface{}{[]interface{}{1, 2, 3}}, "[1 2 3]"},
		{[]interface{}{[]interface{}{"1", "2", "3"}}, "[1 2 3]"},
	}
	for _, c := range cases {
		if key := ToStringKey(c.values...); key != c.key {
			t.Errorf("%v: expected %v, got %v", c.values, c.key, key)
		}
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		elems  interface{}
		elem   interface{}
		target bool
	}{
		{elems: []string{"INSERT", "VALUES", "ON CONFLICT", "RETURNING"}, elem: "RETURNING", target: true},
		{elems: []string{"INSERT", "VALUES", "ON CONFLICT"}, elem: "RETURNING", target: false},
		{elems: []int{1, 2, 3}, elem: 1, target: true},
		{elems: []interface{}{1, 2.0, "3"}, elem: 2.0, target: true},
	}

	for _, test := range tests {
		exists := Contains(test.elems, test.elem)
		if exists != test.target {
			t.Errorf("%v not exist %v", test.elems, test.elem)
		}
	}
}
