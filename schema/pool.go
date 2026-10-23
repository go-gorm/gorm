package schema

import (
	"reflect"
	"sync"
)

// sync pools
var (
	// normalPool caches one sync.Pool per reflection type used for ordinary field scan
	// destinations.
	normalPool sync.Map
	// poolInitializer returns the shared pool for reflectType, allocating
	// reflect.New(reflectType) on a miss. Field setup passes a pointer type, so ordinary
	// scan destinations are pointers to pointers that preserve SQL NULL information.
	poolInitializer = func(reflectType reflect.Type) FieldNewValuePool {
		v, _ := normalPool.LoadOrStore(reflectType, &sync.Pool{
			New: func() interface{} {
				return reflect.New(reflectType).Interface()
			},
		})
		return v.(FieldNewValuePool)
	}
)
