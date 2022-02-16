package schema

import (
	"reflect"
	"sync"
	"time"
)

// sync pools
var (
	normalPool sync.Map
	stringPool = &sync.Pool{
		New: func() interface{} {
			var v string
			ptrV := &v
			return &ptrV
		},
	}
	intPool = &sync.Pool{
		New: func() interface{} {
			var v int64
			ptrV := &v
			return &ptrV
		},
	}
	uintPool = &sync.Pool{
		New: func() interface{} {
			var v uint64
			ptrV := &v
			return &ptrV
		},
	}
	floatPool = &sync.Pool{
		New: func() interface{} {
			var v float64
			ptrV := &v
			return &ptrV
		},
	}
	boolPool = &sync.Pool{
		New: func() interface{} {
			var v bool
			ptrV := &v
			return &ptrV
		},
	}
	timePool = &sync.Pool{
		New: func() interface{} {
			var v time.Time
			ptrV := &v
			return &ptrV
		},
	}
	poolInitializer = func(reflectType reflect.Type) FieldNewValuePool {
		v, _ := normalPool.LoadOrStore(reflectType, &sync.Pool{
			New: func() interface{} {
				return reflect.New(reflectType).Interface()
			},
		})
		return v.(FieldNewValuePool)
	}
)
