// Package utils provides internal utility functions for GORM.
package utils

import (
	"database/sql/driver"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"unicode"
)

var gormSourceDir string

func init() {
	_, file, _, _ := runtime.Caller(0)
	// compatible solution to get gorm source directory with various operating systems
	gormSourceDir = sourceDir(file)
}

func sourceDir(file string) string {
	dir := filepath.Dir(file)
	dir = filepath.Dir(dir)

	s := filepath.Dir(dir)
	if filepath.Base(s) != "gorm.io" {
		s = dir
	}
	return filepath.ToSlash(s) + "/"
}

// CallerFrame retrieves the first relevant stack frame outside of GORM's internal implementation files.
// It skips:
//   - GORM's core source files (identified by gormSourceDir prefix)
//   - Exclude test files (*_test.go)
//   - go-gorm/gen's Generated files (*.gen.go)
func CallerFrame() runtime.Frame {
	// Preserve the original CallerFrame stack depth after introducing callerFrame().
	return callerFrame(4)
}

func callerFrame(skip int) runtime.Frame {
	pcs := [13]uintptr{}
	// skip is caller-path sensitive and should be selected by each public helper.
	len := runtime.Callers(skip, pcs[:])
	frames := runtime.CallersFrames(pcs[:len])
	for i := 0; i < len; i++ {
		// second return value is "more", not "ok"
		frame, _ := frames.Next()
		if (!strings.HasPrefix(frame.File, gormSourceDir) ||
			strings.HasSuffix(frame.File, "_test.go")) && !strings.HasSuffix(frame.File, ".gen.go") {
			return frame
		}
	}

	return runtime.Frame{}
}

// FileWithLineNum return the file name and line number of the current file
func FileWithLineNum() string {
	// Keep FileWithLineNum one frame outer than CallerFrame to preserve pre-v1.31.1 semantics.
	frame := callerFrame(4)
	if frame.PC != 0 {
		return string(strconv.AppendInt(append([]byte(frame.File), ':'), int64(frame.Line), 10))
	}

	return ""
}

func IsInvalidDBNameChar(c rune) bool {
	return !unicode.IsLetter(c) && !unicode.IsNumber(c) && c != '.' && c != '*' && c != '_' && c != '$' && c != '@'
}

// CheckTruth check string true or not
func CheckTruth(vals ...string) bool {
	for _, val := range vals {
		if val != "" && !strings.EqualFold(val, "false") {
			return true
		}
	}
	return false
}

func ToStringKey(values ...interface{}) string {
	if len(values) == 1 {
		return toKeyPart(values[0])
	}

	results := make([]string, len(values))
	for idx, value := range values {
		results[idx] = toKeyPart(value)
	}

	return strings.Join(results, "_")
}

// toKeyPart formats a single ToStringKey component. Signed and sized
// integers take a strconv fast path; zero values of those types map to "nil"
// to match the reflection-based fallback.
func toKeyPart(value interface{}) string {
	if valuer, ok := value.(driver.Valuer); ok {
		value, _ = valuer.Value()
	}

	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	}

	if key, ok := intKeyPart(value); ok {
		return key
	}

	vv := reflect.ValueOf(value)
	if vv.IsValid() && !vv.IsZero() {
		return fmt.Sprint(reflect.Indirect(vv).Interface())
	}
	return "nil"
}

func intKeyPart(value interface{}) (string, bool) {
	var n int64
	switch v := value.(type) {
	case int:
		n = int64(v)
	case int8:
		n = int64(v)
	case int16:
		n = int64(v)
	case int32:
		n = int64(v)
	case int64:
		n = v
	default:
		return uintKeyPart(value)
	}

	if n == 0 {
		return "nil", true
	}
	return strconv.FormatInt(n, 10), true
}

func uintKeyPart(value interface{}) (string, bool) {
	var n uint64
	switch v := value.(type) {
	case uint8:
		n = uint64(v)
	case uint16:
		n = uint64(v)
	case uint32:
		n = uint64(v)
	case uint64:
		n = v
	default:
		return "", false
	}

	if n == 0 {
		return "nil", true
	}
	return strconv.FormatUint(n, 10), true
}

func Contains(elems []string, elem string) bool {
	for _, e := range elems {
		if elem == e {
			return true
		}
	}
	return false
}

func AssertEqual(x, y interface{}) bool {
	if reflect.DeepEqual(x, y) {
		return true
	}
	if x == nil || y == nil {
		return false
	}

	xval := reflect.ValueOf(x)
	yval := reflect.ValueOf(y)
	if xval.Kind() == reflect.Ptr && xval.IsNil() ||
		yval.Kind() == reflect.Ptr && yval.IsNil() {
		return false
	}

	if valuer, ok := x.(driver.Valuer); ok {
		x, _ = valuer.Value()
	}
	if valuer, ok := y.(driver.Valuer); ok {
		y, _ = valuer.Value()
	}
	return reflect.DeepEqual(x, y)
}

func ToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	}
	return ""
}

const nestedRelationSplit = "__"

// NestedRelationName nested relationships like `Manager__Company`
func NestedRelationName(prefix, name string) string {
	return prefix + nestedRelationSplit + name
}

// SplitNestedRelationName Split nested relationships to `[]string{"Manager","Company"}`
func SplitNestedRelationName(name string) []string {
	return strings.Split(name, nestedRelationSplit)
}

// JoinNestedRelationNames nested relationships like `Manager__Company`
func JoinNestedRelationNames(relationNames []string) string {
	return strings.Join(relationNames, nestedRelationSplit)
}

// RTrimSlice Right trims the given slice by given length
func RTrimSlice[T any](v []T, trimLen int) []T {
	if trimLen >= len(v) { // trimLen greater than slice len means fully sliced
		return v[:0]
	}
	if trimLen < 0 { // negative trimLen is ignored
		return v[:]
	}
	return v[:len(v)-trimLen]
}
