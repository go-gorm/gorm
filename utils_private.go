package gorm

import (
	"database/sql"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strings"
)

func fileWithLineNum() string {
	for i := 1; i < 15; i++ {
		_, file, line, ok := runtime.Caller(i)
		if ok && (!regexp.MustCompile(`jinzhu/gorm/.*.go`).MatchString(file) || regexp.MustCompile(`jinzhu/gorm/.*test.go`).MatchString(file)) {
			return fmt.Sprintf("%v:%v", strings.TrimPrefix(file, os.Getenv("GOPATH")+"src/"), line)
		}
	}
	return ""
}

func setFieldValue(field reflect.Value, value interface{}) (result bool) {
	result = false
	if field.IsValid() && field.CanAddr() {
		result = true
		if scanner, ok := field.Addr().Interface().(sql.Scanner); ok {
			scanner.Scan(value)
		} else if reflect.TypeOf(value).ConvertibleTo(field.Type()) {
			field.Set(reflect.ValueOf(value).Convert(field.Type()))
		} else {
			result = false
		}
	}
	return
}

func isBlank(value reflect.Value) bool {
	return reflect.DeepEqual(value.Interface(), reflect.Zero(value.Type()).Interface())
}

func toSearchableMap(attrs ...interface{}) (result interface{}) {
	if len(attrs) > 1 {
		if str, ok := attrs[0].(string); ok {
			result = map[string]interface{}{str: attrs[1]}
		}
	} else if len(attrs) == 1 {
		if attr, ok := attrs[0].(map[string]interface{}); ok {
			result = attr
		}

		if attr, ok := attrs[0].(interface{}); ok {
			result = attr
		}
	}
	return
}

func convertInterfaceToMap(values interface{}) map[string]interface{} {
	attrs := map[string]interface{}{}

	switch value := values.(type) {
	case map[string]interface{}:
		for k, v := range value {
			attrs[ToSnake(k)] = v
		}
	case []interface{}:
		for _, v := range value {
			for key, value := range convertInterfaceToMap(v) {
				attrs[key] = value
			}
		}
	case interface{}:
		reflectValue := reflect.ValueOf(values)

		switch reflectValue.Kind() {
		case reflect.Map:
			for _, key := range reflectValue.MapKeys() {
				attrs[ToSnake(key.Interface().(string))] = reflectValue.MapIndex(key).Interface()
			}
		default:
			scope := Scope{Value: values}
			for _, field := range scope.Fields() {
				if !field.IsBlank {
					attrs[field.DBName] = field.Value
				}
			}
		}
	}
	return attrs
}
