package logger

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"time"
	"unicode"
)

var (
	sqlRegexp                = regexp.MustCompile(`\?`)
	numericPlaceHolderRegexp = regexp.MustCompile(`\$\d+`)
)

func isPrintable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

func SQLFormatter(values ...interface{}) (messages []interface{}) {
	if len(values) > 1 {
		var (
			sql             string
			formattedValues []string
			currentTime     = time.Now().Format("2006-01-02 15:04:05")
		)

		messages = []interface{}{currentTime}

		// duration
		messages = append(messages, fmt.Sprintf(" [%.2fms]", float64(values[0].(time.Duration).Nanoseconds()/1e4)/100.0))

		for _, value := range values[2].([]interface{}) {
			indirectValue := reflect.Indirect(reflect.ValueOf(value))
			if indirectValue.IsValid() {
				value = indirectValue.Interface()
				if t, ok := value.(time.Time); ok {
					formattedValues = append(formattedValues, fmt.Sprintf("'%v'", t.Format("2006-01-02 15:04:05")))
				} else if b, ok := value.([]byte); ok {
					if str := string(b); isPrintable(str) {
						formattedValues = append(formattedValues, fmt.Sprintf("'%v'", str))
					} else {
						formattedValues = append(formattedValues, "'<binary>'")
					}
				} else if r, ok := value.(driver.Valuer); ok {
					if value, err := r.Value(); err == nil && value != nil {
						formattedValues = append(formattedValues, fmt.Sprintf("'%v'", value))
					} else {
						formattedValues = append(formattedValues, "NULL")
					}
				} else {
					formattedValues = append(formattedValues, fmt.Sprintf("'%v'", value))
				}
			} else {
				formattedValues = append(formattedValues, "NULL")
			}
		}

		// differentiate between $n placeholders or else treat like ?
		if numericPlaceHolderRegexp.MatchString(values[1].(string)) {
			sql = values[1].(string)
			for index, value := range formattedValues {
				placeholder := fmt.Sprintf(`\$%d([^\d]|$)`, index+1)
				sql = regexp.MustCompile(placeholder).ReplaceAllString(sql, value+"$1")
			}
		} else {
			formattedValuesLength := len(formattedValues)
			for index, value := range sqlRegexp.Split(values[1].(string), -1) {
				sql += value
				if index < formattedValuesLength {
					sql += formattedValues[index]
				}
			}
		}

		messages = append(messages, sql)
		messages = append(messages, fmt.Sprintf(" \n[%v]", strconv.FormatInt(values[3].(int64), 10)+" rows affected or returned "))
	}

	return
}
