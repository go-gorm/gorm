package gorm

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func assertCallbacks(funcs []func(*DB), fnames []string) (result bool, msg string) {
	var got []string

	for _, f := range funcs {
		got = append(got, getFuncName(f))
	}

	return fmt.Sprint(got) == fmt.Sprint(fnames), fmt.Sprintf("expects %v, got %v", fnames, got)
}

func getFuncName(fc func(*DB)) string {
	fnames := strings.Split(runtime.FuncForPC(reflect.ValueOf(fc).Pointer()).Name(), ".")
	return fnames[len(fnames)-1]
}

func c1(*DB) {}
func c2(*DB) {}
func c3(*DB) {}
func c4(*DB) {}
func c5(*DB) {}

func TestCallbacks(t *testing.T) {
	type callback struct {
		name    string
		before  string
		after   string
		remove  bool
		replace bool
		err     error
		match   func(*DB) bool
		h       func(*DB)
	}

	datas := []struct {
		callbacks []callback
		results   []string
	}{
		{
			callbacks: []callback{{h: c1}, {h: c2}, {h: c3}, {h: c4}, {h: c5}},
			results:   []string{"c1", "c2", "c3", "c4", "c5"},
		},
		{
			callbacks: []callback{{h: c1}, {h: c2}, {h: c3}, {h: c4}, {h: c5, before: "c4"}},
			results:   []string{"c1", "c2", "c3", "c5", "c4"},
		},
		{
			callbacks: []callback{{h: c1}, {h: c2}, {h: c3}, {h: c4, after: "c5"}, {h: c5}},
			results:   []string{"c1", "c2", "c3", "c5", "c4"},
		},
		{
			callbacks: []callback{{h: c1}, {h: c2}, {h: c3}, {h: c4, after: "c5"}, {h: c5, before: "c4"}},
			results:   []string{"c1", "c2", "c3", "c5", "c4"},
		},
		{
			callbacks: []callback{{h: c1}, {h: c2, before: "c4", after: "c5"}, {h: c3}, {h: c4}, {h: c5}},
			results:   []string{"c1", "c5", "c2", "c3", "c4"},
		},
		{
			callbacks: []callback{{h: c1, before: "c3", after: "c4"}, {h: c2, before: "c4", after: "c5"}, {h: c3, before: "c5"}, {h: c4}, {h: c5}},
			results:   []string{"c1", "c3", "c5", "c2", "c4"},
		},
		{
			callbacks: []callback{{h: c1}, {h: c2, before: "c4", after: "c5"}, {h: c3}, {h: c4}, {h: c5}, {h: c2, remove: true}},
			results:   []string{"c1", "c5", "c3", "c4"},
		},
		{
			callbacks: []callback{{h: c1}, {name: "c", h: c2}, {h: c3}, {name: "c", h: c4, replace: true}},
			results:   []string{"c1", "c4", "c3"},
		},
	}

	// func TestRegisterCallbackWithComplexOrder(t *testing.T) {
	// 	var callback2 = &Callback{logger: defaultLogger}

	// 	callback2.Delete().Before("after_create1").After("before_create1").Register("create", create)
	// 	callback2.Delete().Before("create").Register("before_create1", beforeCreate1)
	// 	callback2.Delete().After("before_create1").Register("before_create2", beforeCreate2)
	// 	callback2.Delete().Register("after_create1", afterCreate1)
	// 	callback2.Delete().After("after_create1").Register("after_create2", afterCreate2)

	// 	if !equalFuncs(callback2.deletes, []string{"beforeCreate1", "beforeCreate2", "create", "afterCreate1", "afterCreate2"}) {
	// 		t.Errorf("register callback with order")
	// 	}
	// }

	for idx, data := range datas {
		callbacks := &Callbacks{}

		for _, c := range data.callbacks {
			p := callbacks.Create()

			if c.name == "" {
				c.name = getFuncName(c.h)
			}

			if c.before != "" {
				p = p.Before(c.before)
			}

			if c.after != "" {
				p = p.After(c.after)
			}

			if c.match != nil {
				p = p.Match(c.match)
			}

			if c.remove {
				p.Remove(c.name)
			} else if c.replace {
				p.Replace(c.name, c.h)
			} else {
				p.Register(c.name, c.h)
			}
		}

		if ok, msg := assertCallbacks(callbacks.creates, data.results); !ok {
			t.Errorf("callbacks tests #%v failed, got %v", idx+1, msg)
		}
	}
}
