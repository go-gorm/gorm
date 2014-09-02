package gorm

import (
	"bytes"
	"reflect"
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

func FieldValueByName(name string, value interface{}, withAddr ...bool) (interface{}, bool) {
	data := reflect.Indirect(reflect.ValueOf(value))
	name = SnakeToUpperCamel(name)

	if data.Kind() == reflect.Struct {
		if field := data.FieldByName(name); field.IsValid() {
			if len(withAddr) > 0 && field.CanAddr() {
				return field.Addr().Interface(), true
			} else {
				return field.Interface(), true
			}
		}
	} else if data.Kind() == reflect.Slice {
		elem := data.Type().Elem()
		if elem.Kind() == reflect.Ptr {
			return nil, reflect.New(data.Type().Elem().Elem()).Elem().FieldByName(name).IsValid()
		} else {
			return nil, reflect.New(data.Type().Elem()).Elem().FieldByName(name).IsValid()
		}
	}
	return nil, false
}

func newSafeMap() *safeMap {
	return &safeMap{l: new(sync.RWMutex), m: make(map[string]string)}
}

var smap = newSafeMap()
var umap = newSafeMap()

func ToSnake(u string) string {
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
	smap.Set(u, s)
	return s
}

func SnakeToUpperCamel(s string) string {
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
	umap.Set(s, u)
	return u
}

func parseTagSetting(str string) map[string]string {
	tags := strings.Split(str, ";")
	setting := map[string]string{}
	for _, value := range tags {
		v := strings.Split(value, ":")
		k := strings.TrimSpace(strings.ToUpper(v[0]))
		if len(v) == 2 {
			setting[k] = v[1]
		} else {
			setting[k] = k
		}
	}
	return setting
}
