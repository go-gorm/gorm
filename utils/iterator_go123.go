//go:build go1.23
// +build go1.23

package utils

import (
	"reflect"
)

// isIteratorSeq checks if the given reflect.Value is an iter.Seq[T]
func isIteratorSeq(v reflect.Value) bool {
	if !v.IsValid() {
		return false
	}

	t := v.Type()
	if t.Kind() != reflect.Func {
		return false
	}

	// Check if it's a function with the signature: func(func(T) bool)
	if t.NumIn() != 1 || t.NumOut() != 0 {
		return false
	}

	// Check if the parameter is func(T) bool
	paramType := t.In(0)
	if paramType.Kind() != reflect.Func {
		return false
	}

	if paramType.NumIn() != 1 || paramType.NumOut() != 1 {
		return false
	}

	// Check if the return type is bool
	if paramType.Out(0) != reflect.TypeOf(true) {
		return false
	}

	return true
}

// convertIteratorSeqToSlice converts an iter.Seq[T] to a slice of T
func convertIteratorSeqToSlice(v reflect.Value) reflect.Value {
	// Get the element type from the yield function parameter
	yieldFuncType := v.Type().In(0)
	elemType := yieldFuncType.In(0)

	// Create a slice type for the elements
	sliceType := reflect.SliceOf(elemType)
	result := reflect.MakeSlice(sliceType, 0, 0)

	// Create the yield function that appends to our slice
	yieldFunc := reflect.MakeFunc(yieldFuncType, func(args []reflect.Value) []reflect.Value {
		if len(args) > 0 {
			result = reflect.Append(result, args[0])
		}
		// Always return true to continue iteration
		return []reflect.Value{reflect.ValueOf(true)}
	})

	// Call the iterator with our yield function
	v.Call([]reflect.Value{yieldFunc})

	return result
}

// ConvertIteratorToSlice converts iter.Seq[T] to the appropriate slice []T
func ConvertIteratorToSlice(v reflect.Value) (reflect.Value, bool) {
	if isIteratorSeq(v) {
		return convertIteratorSeqToSlice(v), true
	}

	return v, false
}
