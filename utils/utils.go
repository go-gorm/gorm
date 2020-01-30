package utils

import (
	"fmt"
	"regexp"
	"runtime"
)

var goSrcRegexp = regexp.MustCompile(`jinzhu/gorm/.*.go`)
var goTestRegexp = regexp.MustCompile(`jinzhu/gorm/.*test.go`)

func FileWithLineNum() string {
	for i := 2; i < 15; i++ {
		_, file, line, ok := runtime.Caller(i)
		if ok && (!goSrcRegexp.MatchString(file) || goTestRegexp.MatchString(file)) {
			return fmt.Sprintf("%v:%v", file, line)
		}
	}
	return ""
}
