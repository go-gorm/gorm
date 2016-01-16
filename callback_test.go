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

func create(s *Scope)        {}
func beforeCreate1(s *Scope) {}
func beforeCreate2(s *Scope) {}
func afterCreate1(s *Scope)  {}
func afterCreate2(s *Scope)  {}

func TestRegisterCallback(t *testing.T) {
	var callbacks = &Callbacks{}

	callbacks.Create().Register("before_create1", beforeCreate1)
	callbacks.Create().Register("before_create2", beforeCreate2)
	callbacks.Create().Register("create", create)
	callbacks.Create().Register("after_create1", afterCreate1)
	callbacks.Create().Register("after_create2", afterCreate2)

	if !equalFuncs(callbacks.creates, []string{"beforeCreate1", "beforeCreate2", "create", "afterCreate1", "afterCreate2"}) {
		t.Errorf("register callback")
	}
}

func TestRegisterCallbackWithOrder(t *testing.T) {
	var callbacks1 = &Callbacks{}
	callbacks1.Create().Register("before_create1", beforeCreate1)
	callbacks1.Create().Register("create", create)
	callbacks1.Create().Register("after_create1", afterCreate1)
	callbacks1.Create().Before("after_create1").Register("after_create2", afterCreate2)
	if !equalFuncs(callbacks1.creates, []string{"beforeCreate1", "create", "afterCreate2", "afterCreate1"}) {
		t.Errorf("register callback with order")
	}

	var callbacks2 = &Callbacks{}

	callbacks2.Update().Register("create", create)
	callbacks2.Update().Before("create").Register("before_create1", beforeCreate1)
	callbacks2.Update().After("after_create2").Register("after_create1", afterCreate1)
	callbacks2.Update().Before("before_create1").Register("before_create2", beforeCreate2)
	callbacks2.Update().Register("after_create2", afterCreate2)

	if !equalFuncs(callbacks2.updates, []string{"beforeCreate2", "beforeCreate1", "create", "afterCreate2", "afterCreate1"}) {
		t.Errorf("register callback with order")
	}
}

func TestRegisterCallbackWithComplexOrder(t *testing.T) {
	var callbacks1 = &Callbacks{}

	callbacks1.Query().Before("after_create1").After("before_create1").Register("create", create)
	callbacks1.Query().Register("before_create1", beforeCreate1)
	callbacks1.Query().Register("after_create1", afterCreate1)

	if !equalFuncs(callbacks1.queries, []string{"beforeCreate1", "create", "afterCreate1"}) {
		t.Errorf("register callback with order")
	}

	var callbacks2 = &Callbacks{}

	callbacks2.Delete().Before("after_create1").After("before_create1").Register("create", create)
	callbacks2.Delete().Before("create").Register("before_create1", beforeCreate1)
	callbacks2.Delete().After("before_create1").Register("before_create2", beforeCreate2)
	callbacks2.Delete().Register("after_create1", afterCreate1)
	callbacks2.Delete().After("after_create1").Register("after_create2", afterCreate2)

	if !equalFuncs(callbacks2.deletes, []string{"beforeCreate1", "beforeCreate2", "create", "afterCreate1", "afterCreate2"}) {
		t.Errorf("register callback with order")
	}
}

func replaceCreate(s *Scope) {}

func TestReplaceCallback(t *testing.T) {
	var callbacks = &Callbacks{}

	callbacks.Create().Before("after_create1").After("before_create1").Register("create", create)
	callbacks.Create().Register("before_create1", beforeCreate1)
	callbacks.Create().Register("after_create1", afterCreate1)
	callbacks.Create().Replace("create", replaceCreate)

	if !equalFuncs(callbacks.creates, []string{"beforeCreate1", "replaceCreate", "afterCreate1"}) {
		t.Errorf("replace callback")
	}
}

func TestRemoveCallback(t *testing.T) {
	var callbacks = &Callbacks{}

	callbacks.Create().Before("after_create1").After("before_create1").Register("create", create)
	callbacks.Create().Register("before_create1", beforeCreate1)
	callbacks.Create().Register("after_create1", afterCreate1)
	callbacks.Create().Remove("create")

	if !equalFuncs(callbacks.creates, []string{"beforeCreate1", "afterCreate1"}) {
		t.Errorf("remove callback")
	}
}
