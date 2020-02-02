package tests

import (
	"reflect"
	"testing"
)

func AssertEqual(t *testing.T, r, e interface{}, names ...string) {
	for _, name := range names {
		got := reflect.Indirect(reflect.ValueOf(r)).FieldByName(name).Interface()
		expects := reflect.Indirect(reflect.ValueOf(e)).FieldByName(name).Interface()

		if !reflect.DeepEqual(got, expects) {
			t.Run(name, func(t *testing.T) {
				t.Errorf("expects: %v, got %v", expects, got)
			})
		}
	}
}
