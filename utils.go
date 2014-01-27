package gorm

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type safeMap struct {
	m map[string]string
	l *sync.RWMutex
}

func (s *safeMap) Set(key string, value string) {
	s.l.Lock()
	defer s.l.Unlock()
	s.m[key] = value
}

func (s *safeMap) Get(key string) string {
	s.l.RLock()
	defer s.l.RUnlock()
	return s.m[key]
}

func newSafeMap() *safeMap {
	return &safeMap{l: new(sync.RWMutex), m: make(map[string]string)}
}

var smap = newSafeMap()
var umap = newSafeMap()

func toSnake(u string) string {
	if v := smap.Get(u); v != "" {
		return v
	}

	buf := bytes.NewBufferString("")
	for i, v := range u {
		if i > 0 && v >= 'A' && v <= 'Z' {
			buf.WriteRune('_')
		}
		buf.WriteRune(v)
	}

	s := strings.ToLower(buf.String())
	go smap.Set(u, s)
	return s
}

func snakeToUpperCamel(s string) string {
	if v := umap.Get(s); v != "" {
		return v
	}

	buf := bytes.NewBufferString("")
	for _, v := range strings.Split(s, "_") {
		if len(v) > 0 {
			buf.WriteString(strings.ToUpper(v[:1]))
			buf.WriteString(v[1:])
		}
	}

	u := buf.String()
	go umap.Set(s, u)
	return u
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

func fileWithLineNum() string {
	for i := 1; i < 15; i++ {
		_, file, line, ok := runtime.Caller(i)
		if ok && (!regexp.MustCompile(`jinzhu/gorm/.*.go`).MatchString(file) || regexp.MustCompile(`jinzhu/gorm/.*test.go`).MatchString(file)) {
			return fmt.Sprintf("%v:%v", strings.TrimPrefix(file, os.Getenv("GOPATH")+"src/"), line)
		}
	}
	return ""
}

func setFieldValue(field reflect.Value, value interface{}) bool {
	if field.IsValid() && field.CanAddr() {
		switch field.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if str, ok := value.(string); ok {
				value, _ = strconv.Atoi(str)
			}
			field.SetInt(reflect.ValueOf(value).Int())
		default:
			if scanner, ok := field.Addr().Interface().(sql.Scanner); ok {
				scanner.Scan(value)
			} else {
				field.Set(reflect.ValueOf(value))
			}
		}
		return true
	}

	return false
}

func isBlank(value reflect.Value) bool {
	return reflect.DeepEqual(value.Interface(), reflect.Zero(value.Type()).Interface())
}

func convertInterfaceToMap(values interface{}) map[string]interface{} {
	attrs := map[string]interface{}{}

	switch value := values.(type) {
	case map[string]interface{}:
		for k, v := range value {
			attrs[toSnake(k)] = v
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
				attrs[toSnake(key.Interface().(string))] = reflectValue.MapIndex(key).Interface()
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
