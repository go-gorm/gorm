package utils

import (
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"unicode"
)

var goSrcRegexp = regexp.MustCompile(`jinzhu/gorm(@.*)?/.*.go`)
var goTestRegexp = regexp.MustCompile(`jinzhu/gorm(@.*)?/.*test.go`)

func FileWithLineNum() string {
	for i := 2; i < 15; i++ {
		_, file, line, ok := runtime.Caller(i)
		if ok && (!goSrcRegexp.MatchString(file) || goTestRegexp.MatchString(file)) {
			return fmt.Sprintf("%v:%v", file, line)
		}
	}
	return ""
}

func IsChar(c rune) bool {
	return !unicode.IsLetter(c) && !unicode.IsNumber(c)
}

func CheckTruth(val interface{}) bool {
	if v, ok := val.(bool); ok {
		return v
	}

	if v, ok := val.(string); ok {
		v = strings.ToLower(v)
		return v != "false"
	}

	return !reflect.ValueOf(val).IsZero()
}

func ToStringKey(values ...reflect.Value) string {
	results := make([]string, len(values))

	for idx, value := range values {
		rv := reflect.Indirect(value).Interface()

		switch v := rv.(type) {
		case string:
			results[idx] = v
		case []byte:
			results[idx] = string(v)
		case uint:
			results[idx] = strconv.FormatUint(uint64(v), 10)
		default:
			results[idx] = fmt.Sprint(v)
		}
	}

	return strings.Join(results, "_")
}
