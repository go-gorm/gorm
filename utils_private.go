package gorm

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strings"
)

func fileWithLineNum() string {
	for i := 2; i < 15; i++ {
		_, file, line, ok := runtime.Caller(i)
		if ok && (!regexp.MustCompile(`jinzhu/gorm/.*.go`).MatchString(file) || regexp.MustCompile(`jinzhu/gorm/.*test.go`).MatchString(file)) {
			return fmt.Sprintf("%v:%v", file, line)
		}
	}
	return ""
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
			attrs[ToDBName(k)] = v
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
				attrs[ToDBName(key.Interface().(string))] = reflectValue.MapIndex(key).Interface()
			}
		default:
			for _, field := range (&Scope{Value: values}).Fields() {
				if !field.IsBlank && !field.IsIgnored {
					attrs[field.DBName] = field.Field.Interface()
				}
			}
		}
	}
	return attrs
}

func equalAsString(a interface{}, b interface{}) bool {
	return toString(a) == toString(b)
}

func toString(str interface{}) string {
	if values, ok := str.([]interface{}); ok {
		var results []string
		for _, value := range values {
			results = append(results, toString(value))
		}
		return strings.Join(results, "_")
	} else if bytes, ok := str.([]byte); ok {
		return string(bytes)
	} else if reflectValue := reflect.Indirect(reflect.ValueOf(str)); reflectValue.IsValid() {
		return fmt.Sprintf("%v", reflectValue.Interface())
	}
	return ""
}

func makeSlice(elemType reflect.Type) interface{} {
	if elemType.Kind() == reflect.Slice {
		elemType = elemType.Elem()
	}
	sliceType := reflect.SliceOf(elemType)
	slice := reflect.New(sliceType)
	slice.Elem().Set(reflect.MakeSlice(sliceType, 0, 0))
	return slice.Interface()
}

func strInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// getValueFromFields return given fields's value
func getValueFromFields(value reflect.Value, fieldNames []string) (results []interface{}) {
	// If value is a nil pointer, Indirect returns a zero Value!
	// Therefor we need to check for a zero value,
	// as FieldByName could panic
	if indirectValue := reflect.Indirect(value); indirectValue.IsValid() {
		for _, fieldName := range fieldNames {
			if fieldValue := indirectValue.FieldByName(fieldName); fieldValue.IsValid() {
				result := fieldValue.Interface()
				if r, ok := result.(driver.Valuer); ok {
					result, _ = r.Value()
				}
				results = append(results, result)
			}
		}
	}
	return
}
