package logger

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

func isPrintable(s []byte) bool {
	for _, r := range s {
		if !unicode.IsPrint(rune(r)) {
			return false
		}
	}
	return true
}

var convertableTypes = []reflect.Type{reflect.TypeOf(time.Time{}), reflect.TypeOf(false), reflect.TypeOf([]byte{})}

func ExplainSQL(sql string, numericPlaceholder *regexp.Regexp, escaper string, avars ...interface{}) string {
	var convertParams func(interface{}, int)
	var vars = make([]interface{}, len(avars))
	copy(vars, avars)

	convertParams = func(v interface{}, idx int) {
		switch v := v.(type) {
		case bool:
			vars[idx] = fmt.Sprint(v)
		case time.Time:
			if v.IsZero() {
				vars[idx] = escaper + "0000-00-00 00:00:00" + escaper
			} else {
				vars[idx] = escaper + v.Format("2006-01-02 15:04:05.999") + escaper
			}
		case []byte:
			if isPrintable(v) {
				vars[idx] = escaper + strings.Replace(string(v), escaper, "\\"+escaper, -1) + escaper
			} else {
				vars[idx] = escaper + "<binary>" + escaper
			}
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			vars[idx] = fmt.Sprintf("%d", v)
		case float64, float32:
			vars[idx] = fmt.Sprintf("%.6f", v)
		case string:
			vars[idx] = escaper + strings.Replace(v, escaper, "\\"+escaper, -1) + escaper
		default:
			rv := reflect.ValueOf(v)
			if v == nil || !rv.IsValid() || rv.Kind() == reflect.Ptr && rv.IsNil() {
				vars[idx] = "NULL"
			} else if valuer, ok := v.(driver.Valuer); ok {
				v, _ = valuer.Value()
				convertParams(v, idx)
			} else if rv.Kind() == reflect.Ptr && !rv.IsZero() {
				convertParams(reflect.Indirect(rv).Interface(), idx)
			} else {
				for _, t := range convertableTypes {
					if rv.Type().ConvertibleTo(t) {
						convertParams(rv.Convert(t).Interface(), idx)
						return
					}
				}
				vars[idx] = escaper + strings.Replace(fmt.Sprint(v), escaper, "\\"+escaper, -1) + escaper
			}
		}
	}

	for idx, v := range vars {
		convertParams(v, idx)
	}

	if numericPlaceholder == nil {
		for _, v := range vars {
			sql = strings.Replace(sql, "?", v.(string), 1)
		}
	} else {
		sql = numericPlaceholder.ReplaceAllString(sql, "$$$1$$")
		for idx, v := range vars {
			sql = strings.Replace(sql, "$"+strconv.Itoa(idx+1)+"$", v.(string), 1)
		}
	}

	return sql
}
