// +build go1.13

package gorm

import "errors"

// use go1.13 errors.Is to check is err is target error.
var isError = errors.Is
