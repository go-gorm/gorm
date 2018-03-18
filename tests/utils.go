package tests

import "fmt"

// Format format test message
type Format func(value interface{}, args ...interface{}) string

// FormatWithMsg format test messages
func FormatWithMsg(msg string) Format {
	return func(value interface{}, args ...interface{}) string {
		return fmt.Sprintf("[%v] %v", msg, fmt.Sprintf(fmt.Sprint(value), args...))
	}
}
