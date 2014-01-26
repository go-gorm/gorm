package gorm

import (
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func equalFuncs(funcs []*func(s *Scope), fnames []string) bool {
	var names []string
	for _, f := range funcs {
		fnames := strings.Split(runtime.FuncForPC(reflect.ValueOf(*f).Pointer()).Name(), ".")
		names = append(names, fnames[len(fnames)-1])
	}
	return reflect.DeepEqual(names, fnames)
}

func create(s *Scope)         {}
func before_create1(s *Scope) {}
func before_create2(s *Scope) {}
func after_create1(s *Scope)  {}
func after_create2(s *Scope)  {}

func TestRegisterCallback(t *testing.T) {
	var callback = &callback{processors: []*callback_processor{}}

	callback.Create().Register("before_create1", before_create1)
	callback.Create().Register("before_create2", before_create2)
	callback.Create().Register("create", create)
	callback.Create().Register("after_create1", after_create1)
	callback.Create().Register("after_create2", after_create2)

	if !equalFuncs(callback.creates, []string{"before_create1", "before_create2", "create", "after_create1", "after_create2"}) {
		t.Errorf("register callback")
	}
}

func TestRegisterCallbackWithBasicOrder(t *testing.T) {
	var callback = &callback{processors: []*callback_processor{}}

	callback.Update().Register("create", create)
	callback.Update().Before("create").Register("before_create1", before_create1)
	callback.Update().After("after_create2").Register("after_create1", after_create1)
	callback.Update().Before("before_create1").Register("before_create2", before_create2)
	callback.Update().Register("after_create2", after_create2)

	if !equalFuncs(callback.updates, []string{"before_create2", "before_create1", "create", "after_create2", "after_create1"}) {
		t.Errorf("register callback with order")
	}
}

func TestRegisterCallbackWithComplexOrder1(t *testing.T) {
	var callback = &callback{processors: []*callback_processor{}}

	callback.Query().Before("after_create1").After("before_create1").Register("create", create)
	callback.Query().Register("before_create1", before_create1)
	callback.Query().Register("after_create1", after_create1)

	if !equalFuncs(callback.queries, []string{"before_create1", "create", "after_create1"}) {
		t.Errorf("register callback with order")
	}
}

func TestRegisterCallbackWithComplexOrder2(t *testing.T) {
	var callback = &callback{processors: []*callback_processor{}}

	callback.Delete().Before("after_create1").After("before_create1").Register("create", create)
	callback.Delete().Before("create").Register("before_create1", before_create1)
	callback.Delete().After("before_create1").Register("before_create2", before_create2)
	callback.Delete().Register("after_create1", after_create1)
	callback.Delete().After("after_create1").Register("after_create2", after_create2)

	if !equalFuncs(callback.deletes, []string{"before_create1", "before_create2", "create", "after_create1", "after_create2"}) {
		t.Errorf("register callback with order")
	}
}

func replace_create(s *Scope) {}

func TestReplaceCallback(t *testing.T) {
	var callback = &callback{processors: []*callback_processor{}}

	callback.Create().Before("after_create1").After("before_create1").Register("create", create)
	callback.Create().Register("before_create1", before_create1)
	callback.Create().Register("after_create1", after_create1)
	callback.Create().Replace("create", replace_create)

	if !equalFuncs(callback.creates, []string{"before_create1", "replace_create", "after_create1"}) {
		t.Errorf("replace callback")
	}
}

func TestRemoveCallback(t *testing.T) {
	var callback = &callback{processors: []*callback_processor{}}

	callback.Create().Before("after_create1").After("before_create1").Register("create", create)
	callback.Create().Register("before_create1", before_create1)
	callback.Create().Register("after_create1", after_create1)
	callback.Create().Remove("create")

	if !equalFuncs(callback.creates, []string{"before_create1", "after_create1"}) {
		t.Errorf("remove callback")
	}
}
