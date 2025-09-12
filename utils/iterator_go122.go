//go:build !go1.23
// +build !go1.23

package utils

import "reflect"

// ConvertIteratorToSlice is a no-op for Go versions < 1.23
func ConvertIteratorToSlice(v reflect.Value) (reflect.Value, bool) {
	return v, false
}
