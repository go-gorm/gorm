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

// gormSourceDir is the normalized source prefix used to skip GORM implementation frames
// while locating callers. It is initialized from this file's runtime location.
var gormSourceDir string

// init records the source-directory prefix used by caller reporting. Path normalization
// accommodates module cache and checkout layouts.
func init() {
	_, file, _, _ := runtime.Caller(0)
	// compatible solution to get gorm source directory with various operating systems
	gormSourceDir = sourceDir(file)
}

// sourceDir derives the slash-normalized GORM source prefix from a source filename,
// accounting for a gorm.io parent directory. It is a stack-filter heuristic, not
// filesystem discovery.
func sourceDir(file string) string {
	dir := filepath.Dir(file)
	dir = filepath.Dir(dir)

	s := filepath.Dir(dir)
	if filepath.Base(s) != "gorm.io" {
		s = dir
	}
	return filepath.ToSlash(s) + "/"
}

// CallerFrame returns the first relevant caller outside GORM implementation files,
// retaining _test.go frames and skipping .gen.go frames. It examines a bounded stack
// window and returns a zero runtime.Frame when no candidate is found. The fixed skip
// count is part of caller-reporting behavior.
func CallerFrame() runtime.Frame {
	// Preserve the original CallerFrame stack depth after introducing callerFrame().
	return callerFrame(4)
}

// callerFrame scans up to 13 runtime caller PCs after skip, excluding GORM
// implementation and generated .gen.go files while retaining test files. It returns a
// zero frame if none qualifies.
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

// FileWithLineNum formats the first relevant external caller as path:line, or an empty
// string if none is found. It uses its own fixed stack skip to preserve historical
// reporting behavior.
func FileWithLineNum() string {
	// Keep FileWithLineNum one frame outer than CallerFrame to preserve pre-v1.31.1 semantics.
	frame := callerFrame(4)
	if frame.PC != 0 {
		return string(strconv.AppendInt(append([]byte(frame.File), ':'), int64(frame.Line), 10))
	}

	return ""
}

// IsInvalidDBNameChar identifies separators for simple identifier heuristics. Letters,
// numbers, dot, star, underscore, dollar, and at-sign are accepted; this is not SQL
// validation or an injection defense.
func IsInvalidDBNameChar(c rune) bool {
	return !unicode.IsLetter(c) && !unicode.IsNumber(c) && c != '.' && c != '*' && c != '_' && c != '$' && c != '@'
}

// CheckTruth reports true when any value is non-empty and not case-insensitively equal
// to "false". Values such as "0" count as true; it is a GORM tag convention, not
// strconv.ParseBool.
func CheckTruth(vals ...string) bool {
	for _, val := range vals {
		if val != "" && !strings.EqualFold(val, "false") {
			return true
		}
	}
	return false
}

// ToStringKey converts identity components into an underscore-joined lookup key,
// unwrapping driver.Valuer values and using special zero/nil formatting. It is intended
// for internal association maps, not collision-free serialization: delimiters and
// textual values can collide.
func ToStringKey(values ...interface{}) string {
	results := make([]string, len(values))

	for idx, value := range values {
		if valuer, ok := value.(driver.Valuer); ok {
			value, _ = valuer.Value()
		}

		switch v := value.(type) {
		case string:
			results[idx] = v
		case []byte:
			results[idx] = string(v)
		case uint:
			results[idx] = strconv.FormatUint(uint64(v), 10)
		default:
			results[idx] = "nil"
			vv := reflect.ValueOf(v)
			if vv.IsValid() && !vv.IsZero() {
				results[idx] = fmt.Sprint(reflect.Indirect(vv).Interface())
			}
		}
	}

	return strings.Join(results, "_")
}

// Contains reports whether elems contains elem using exact, case-sensitive string
// equality. It scans linearly and does not modify the slice.
func Contains(elems []string, elem string) bool {
	for _, e := range elems {
		if elem == e {
			return true
		}
	}
	return false
}

// AssertEqual compares values with reflect.DeepEqual, then compares unwrapped
// driver.Valuer results when necessary. Nil pointers are handled before calling Value;
// conversion errors are ignored. It is a schema/update comparison helper, not arbitrary
// SQL equality.
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

// ToString formats a string or built-in signed/unsigned integer in decimal. Unsupported
// types, including floats and uintptr, return an empty string; it is not fmt.Sprint.
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

// nestedRelationSplit separates components of GORM's nested association column aliases.
// Names containing this delimiter need special care because splitting is purely
// textual.
const nestedRelationSplit = "__"

// NestedRelationName joins prefix and name with the nested-relation delimiter, for
// example Manager__Company. It does not quote or validate SQL identifiers.
func NestedRelationName(prefix, name string) string {
	return prefix + nestedRelationSplit + name
}

// SplitNestedRelationName splits a nested association alias on double underscores. An
// empty input produces one empty component, following strings.Split.
func SplitNestedRelationName(name string) []string {
	return strings.Split(name, nestedRelationSplit)
}

// JoinNestedRelationNames joins association path components with double underscores for
// SQL aliases and scan mapping.
func JoinNestedRelationNames(relationNames []string) string {
	return strings.Join(relationNames, nestedRelationSplit)
}

// RTrimSlice returns a view omitting trimLen elements from the right. A negative value
// leaves the slice length unchanged; trimLen at least the length returns an empty view.
// It shares the original backing array and does not clear removed values or release
// their references.
func RTrimSlice[T any](v []T, trimLen int) []T {
	if trimLen >= len(v) { // trimLen greater than slice len means fully sliced
		return v[:0]
	}
	if trimLen < 0 { // negative trimLen is ignored
		return v[:]
	}
	return v[:len(v)-trimLen]
}
