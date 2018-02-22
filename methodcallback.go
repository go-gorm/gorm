package gorm

import (
	"reflect"
	"fmt"
	"sync"
)

var methodPtrType = reflect.PtrTo(reflect.TypeOf(Method{}))

type safeEnabledFieldTypes struct {
	m map[reflect.Type]bool
	l *sync.RWMutex
}

func newSafeEnabledFieldTypes() safeEnabledFieldTypes {
	return safeEnabledFieldTypes{make(map[reflect.Type]bool), new(sync.RWMutex)}
}

func (s *safeEnabledFieldTypes) Set(key interface{}, enabled bool) {
	switch k := key.(type) {
	case reflect.Type:
		k = indirectType(k)
		s.l.Lock()
		defer s.l.Unlock()
		s.m[k] = enabled
	default:
		s.Set(reflect.TypeOf(key), enabled)
	}
}

func (s *safeEnabledFieldTypes) Get(key interface{}) (enabled bool, ok bool) {
	switch k := key.(type) {
	case reflect.Type:
		k = indirectType(k)
		s.l.RLock()
		defer s.l.RUnlock()
		enabled, ok = s.m[k]
		return
	default:
		return s.Get(reflect.TypeOf(key))
	}
}

func (s *safeEnabledFieldTypes) Has(key interface{}) (ok bool) {
	switch k := key.(type) {
	case reflect.Type:
		k = indirectType(k)
		s.l.RLock()
		defer s.l.RUnlock()
		_, ok = s.m[k]
		return
	default:
		return s.Has(reflect.TypeOf(key))
	}
}

func (s *safeEnabledFieldTypes) Del(key interface{}) (ok bool) {
	switch k := key.(type) {
	case reflect.Type:
		k = indirectType(k)
		s.l.Lock()
		defer s.l.Unlock()
		_, ok = s.m[k]
		if ok {
			delete(s.m, k)
		}
		return
	default:
		return s.Del(reflect.TypeOf(key))
	}
}

type StructFieldMethodCallbacksRegistrator struct {
	Callbacks  map[string]reflect.Value
	FieldTypes safeEnabledFieldTypes
	l          *sync.RWMutex
}

// Register new field type and enable all available callbacks for here
func (registrator *StructFieldMethodCallbacksRegistrator) RegisterFieldType(typs ...interface{}) {
	for _, typ := range typs {
		if !registrator.FieldTypes.Has(typ) {
			registrator.FieldTypes.Set(typ, true)
		}
	}
}

// Unregister field type and return if ok
func (registrator *StructFieldMethodCallbacksRegistrator) UnregisterFieldType(typ interface{}) (ok bool) {
	return registrator.FieldTypes.Del(typ)
}

// Enable all callbacks for field type
func (registrator *StructFieldMethodCallbacksRegistrator) EnableFieldType(typs ...interface{}) {
	for _, typ := range typs {
		registrator.FieldTypes.Set(typ, true)
	}
}

// Disable all callbacks for field type
func (registrator *StructFieldMethodCallbacksRegistrator) DisableFieldType(typs ...interface{}) {
	for _, typ := range typs {
		registrator.FieldTypes.Set(typ, false)
	}
}

// Return if all callbacks for field type is enabled
func (registrator *StructFieldMethodCallbacksRegistrator) IsEnabledFieldType(typ interface{}) bool {
	if enabled, ok := registrator.FieldTypes.Get(typ); ok {
		return enabled
	}
	return false
}

// Return if field type is registered
func (registrator *StructFieldMethodCallbacksRegistrator) RegisteredFieldType(typ interface{}) bool {
	return registrator.FieldTypes.Has(typ)
}

// Register new callback for fields have method methodName
func (registrator *StructFieldMethodCallbacksRegistrator) registerCallback(methodName string, caller interface{}) error {
	value := indirect(reflect.ValueOf(caller))

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

	if value.Type().In(1).Kind() != reflect.Interface {
		return fmt.Errorf("Second arg of caller %v for method %q isn't a interface{} type.", value.Type(), methodName)
	}

	registrator.l.Lock()
	defer registrator.l.Unlock()
	registrator.Callbacks[methodName] = value
	return nil
}

// Register many callbacks where key is the methodName and value is a caller function.
func (registrator *StructFieldMethodCallbacksRegistrator) registerCallbackMany(items ...map[string]interface{}) error {
	for i, m := range items {
		for methodName, callback := range m {
			err := registrator.registerCallback(methodName, callback)
			if err != nil {
				return fmt.Errorf("Register arg[%v][%q] failed: %v", i, methodName, err)
			}
		}
	}
	return nil
}

func NewStructFieldMethodCallbacksRegistrator() *StructFieldMethodCallbacksRegistrator {
	return &StructFieldMethodCallbacksRegistrator{make(map[string]reflect.Value), newSafeEnabledFieldTypes(),
		new(sync.RWMutex)}
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
	checkOrPanic(StructFieldMethodCallbacks.registerCallbackMany(map[string]interface{}{
		"AfterScan": AfterScanMethodCallback,
	}))
}