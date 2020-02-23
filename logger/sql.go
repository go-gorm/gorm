package logger

import (
	"database/sql/driver"
	"fmt"
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

func ExplainSQL(sql string, numericPlaceholder *regexp.Regexp, escaper string, vars ...interface{}) string {
	for idx, v := range vars {
		if valuer, ok := v.(driver.Valuer); ok {
			v, _ = valuer.Value()
		}

		switch v := v.(type) {
		case bool:
			vars[idx] = fmt.Sprint(v)
		case time.Time:
			vars[idx] = escaper + v.Format("2006-01-02 15:04:05") + escaper
		case *time.Time:
			vars[idx] = escaper + v.Format("2006-01-02 15:04:05") + escaper
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
			if v == nil {
				vars[idx] = "NULL"
			} else {
				vars[idx] = escaper + strings.Replace(fmt.Sprint(v), escaper, "\\"+escaper, -1) + escaper
			}
		}
	}

	if numericPlaceholder == nil {
		for _, v := range vars {
			sql = strings.Replace(sql, "?", v.(string), 1)
		}
	} else {
		sql = numericPlaceholder.ReplaceAllString(sql, "$$$$$1")
		for idx, v := range vars {
			sql = strings.Replace(sql, "$$"+strconv.Itoa(idx), v.(string), 1)
		}
	}

	return sql
}
