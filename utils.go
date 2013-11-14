package gorm

import (
	"bytes"
	"database/sql"

	"errors"
	"reflect"
	"strconv"

	"fmt"
	"strings"
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

func getInterfaceAsString(value interface{}) (str string, err error) {
	switch value := value.(type) {
	case string:
		str = value
	case int:
		if value < 0 {
			str = ""
		} else {
			str = strconv.Itoa(value)
		}
	default:
		err = errors.New(fmt.Sprintf("Can't understand %v", value))
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
