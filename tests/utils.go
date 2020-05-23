package tests

import (
	"reflect"
	"testing"
	"time"
)

func AssertObjEqual(t *testing.T, r, e interface{}, names ...string) {
	for _, name := range names {
		got := reflect.Indirect(reflect.ValueOf(r)).FieldByName(name).Interface()
		expect := reflect.Indirect(reflect.ValueOf(e)).FieldByName(name).Interface()
		t.Run(name, func(t *testing.T) {
			AssertEqual(t, got, expect)
		})
	}
}

func AssertEqual(t *testing.T, got, expect interface{}) {
	if !reflect.DeepEqual(got, expect) {
		isEqual := func() {
			if curTime, ok := got.(time.Time); ok {
				format := "2006-01-02T15:04:05Z07:00"
				if curTime.Format(format) != expect.(time.Time).Format(format) {
					t.Errorf("expect: %v, got %v", expect.(time.Time).Format(format), curTime.Format(format))
				}
			} else if got != expect {
				t.Errorf("expect: %#v, got %#v", expect, got)
			}
		}

		if got == expect {
			return
		}

		if reflect.Indirect(reflect.ValueOf(got)).IsValid() != reflect.Indirect(reflect.ValueOf(expect)).IsValid() {
			t.Errorf("expect: %+v, got %+v", expect, got)
			return
		}

		if got != nil {
			got = reflect.Indirect(reflect.ValueOf(got)).Interface()
		}

		if expect != nil {
			expect = reflect.Indirect(reflect.ValueOf(expect)).Interface()
		}

		if reflect.ValueOf(got).Type().ConvertibleTo(reflect.ValueOf(expect).Type()) {
			got = reflect.ValueOf(got).Convert(reflect.ValueOf(expect).Type()).Interface()
			isEqual()
		} else if reflect.ValueOf(expect).Type().ConvertibleTo(reflect.ValueOf(got).Type()) {
			expect = reflect.ValueOf(got).Convert(reflect.ValueOf(got).Type()).Interface()
			isEqual()
		}
	}
}
