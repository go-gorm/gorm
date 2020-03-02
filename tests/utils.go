package tests

import (
	"reflect"
	"testing"
	"time"
)

func AssertEqual(t *testing.T, r, e interface{}, names ...string) {
	for _, name := range names {
		got := reflect.Indirect(reflect.ValueOf(r)).FieldByName(name).Interface()
		expects := reflect.Indirect(reflect.ValueOf(e)).FieldByName(name).Interface()

		if !reflect.DeepEqual(got, expects) {
			got = reflect.Indirect(reflect.ValueOf(got)).Interface()
			expects = reflect.Indirect(reflect.ValueOf(got)).Interface()
			if curTime, ok := got.(time.Time); ok {
				format := "2006-01-02T15:04:05Z07:00"
				if curTime.Format(format) != expects.(time.Time).Format(format) {
					t.Errorf("expects: %v, got %v", expects.(time.Time).Format(format), curTime.Format(format))
				}
			} else {
				t.Run(name, func(t *testing.T) {
					t.Errorf("expects: %v, got %v", expects, got)
				})
			}
		}
	}
}
