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
