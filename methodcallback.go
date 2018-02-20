package gorm

import (
	"reflect"
	"fmt"
)

var interfaceType = reflect.TypeOf(func(a interface{}) {}).In(0)
var methodPtrType = reflect.PtrTo(reflect.TypeOf(Method{}))

type StructFieldMethodCallbacksRegistrator struct {
	Callbacks map[string]reflect.Value
}

func (registrator *StructFieldMethodCallbacksRegistrator) Register(methodName string, caller interface{}) error {
	value := reflect.ValueOf(caller)

	if value.Kind() != reflect.Func {
		return fmt.Errorf("Caller of method %q isn't a function.", methodName)
	}

	if value.Type().NumIn() < 2 {
		return fmt.Errorf("The caller function %v for method %q require two args. Example: func(methodInfo *gorm.Method, method interface{}).",
			value.Type(), methodName)
	}

	if value.Type().In(0) != methodPtrType {
		return fmt.Errorf("First arg of caller %v for method %q isn't a %v type.", value.Type(), methodName, methodPtrType)
	}

	if value.Type().In(1) != interfaceType {
		return fmt.Errorf("Second arg of caller %v for method %q isn't a interface{} type.", value.Type(), methodName)
	}

	registrator.Callbacks[methodName] = value
	return nil
}

func (registrator *StructFieldMethodCallbacksRegistrator) RegisterMany(items ...map[string]interface{}) error {
	for i, m := range items {
		for methodName, callback := range m {
			err := registrator.Register(methodName, callback)
			if err != nil {
				return fmt.Errorf("Register arg[%v][%q] failed: %v", i, methodName, err)
			}
		}
	}
	return nil
}

func NewStructFieldMethodCallbacksRegistrator() *StructFieldMethodCallbacksRegistrator {
	return &StructFieldMethodCallbacksRegistrator{make(map[string]reflect.Value)}
}

func AfterScanMethodCallback(methodInfo *Method, method interface{}, field *Field, scope *Scope) {
	switch method := method.(type) {
	case func():
		method()
	case func(*Scope):
		method(scope)
	case func(*Scope, *Field):
		method(scope, field)
	case func(*DB, *Field):
		newDB := scope.NewDB()
		method(newDB, field)
		scope.Err(newDB.Error)
	case func() error:
		scope.Err(method())
	case func(*Scope) error:
		scope.Err(method(scope))
	case func(*Scope, *Field) error:
		scope.Err(method(scope, field))
	case func(*DB) error:
		newDB := scope.NewDB()
		scope.Err(method(newDB))
		scope.Err(newDB.Error)
	default:
		scope.Err(fmt.Errorf("Invalid AfterScan method callback %v of type %v", reflect.ValueOf(method).Type(), field.Struct.Type))
	}
}

// StructFieldMethodCallbacks is a default registrator for model fields callbacks where the field type is a struct
// and have a callback method.
// Default methods callbacks:
// AfterScan: Call method `AfterScanMethodCallback` after scan field from sql row.
var StructFieldMethodCallbacks = NewStructFieldMethodCallbacksRegistrator()

func init() {
	checkOrPanic(StructFieldMethodCallbacks.RegisterMany(map[string]interface{}{
		"AfterScan": AfterScanMethodCallback,
	}))
}
