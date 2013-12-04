package gorm

import (
	"bytes"
	"database/sql"
	"reflect"
	"strconv"
	"strings"
)

var toSnakeMap = map[string]string{}
var toUpperMap = map[string]string{}

func toSnake(u string) string {
	if v := toSnakeMap[u]; v != "" {
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
	toSnakeMap[u] = s
	return s
}

func snakeToUpperCamel(s string) string {
	if v := toUpperMap[s]; v != "" {
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
	toUpperMap[s] = u
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

func setFieldValue(field reflect.Value, value interface{}) bool {
	if field.IsValid() && field.CanAddr() {
		switch field.Kind() {
		case reflect.Int, reflect.Int32, reflect.Int64:
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
