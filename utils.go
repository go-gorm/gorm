package gorm

import (
	"bytes"
	"database/sql"
	"fmt"

	"reflect"
	"strconv"
	"strings"
	"time"
)

func toSnake(s string) string {
	buf := bytes.NewBufferString("")
	for i, v := range s {
		if i > 0 && v >= 'A' && v <= 'Z' {
			buf.WriteRune('_')
		}
		buf.WriteRune(v)
	}
	return strings.ToLower(buf.String())
}

func snakeToUpperCamel(s string) string {
	buf := bytes.NewBufferString("")
	for _, v := range strings.Split(s, "_") {
		if len(v) > 0 {
			buf.WriteString(strings.ToUpper(v[:1]))
			buf.WriteString(v[1:])
		}
	}
	return buf.String()
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
	switch value.Kind() {
	case reflect.Int, reflect.Int64, reflect.Int32:
		return value.Int() == 0
	case reflect.String:
		return value.String() == ""
	case reflect.Slice:
		return value.Len() == 0
	case reflect.Struct:
		time_value, is_time := value.Interface().(time.Time)
		if is_time {
			return time_value.IsZero()
		} else {
			_, is_scanner := reflect.New(value.Type()).Interface().(sql.Scanner)
			if is_scanner {
				return !value.FieldByName("Valid").Interface().(bool)
			} else {
				m := &Model{data: value.Interface()}
				fields := m.columnsHasValue("other")
				if len(fields) == 0 {
					return true
				}
			}
		}
	}
	return false
}

func debug(values ...interface{}) {
	fmt.Println("*****************")
	fmt.Println(values)
}
