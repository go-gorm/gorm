package gorm

import (
	"database/sql/driver"
	"fmt"
	"log"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"time"
	"unicode"
)

const (
	// RESET is the shell code to disable output color
	RESET = "\033[0m"
	// CYAN is the shell code for CYAN output color
	CYAN = "\033[36;1m"
	// MAGENTA is the shell code for MAGENTA output color
	MAGENTA = "\033[35m"
	// RED is the shell code for RED output color
	RED = "\033[36;31m"
	// REDBOLD is the shell code for REDBOLD output color
	REDBOLD = "\033[31;1m"
	// YELLOW is the shell code for YELLOW output color
	YELLOW = "\033[33m"
)

var (
	defaultLogger            = Logger{log.New(os.Stdout, "\r\n", 0)}
	sqlRegexp                = regexp.MustCompile(`\?`)
	numericPlaceHolderRegexp = regexp.MustCompile(`\$\d+`)
	reset                    = ""
	cyan                     = ""
	magenta                  = ""
	red                      = ""
	redBold                  = ""
	yellow                   = ""
)

func isPrintable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

// NoColor checks for environment if color should be enabled
func NoColor() bool {
	// https://no-color.org/
	return os.Getenv("NO_COLOR") != ""
}

// LogFormatter is a default logger with timestamps, sql error handling, loglevel and NO_COLOR support
var LogFormatter = func(values ...interface{}) (messages []interface{}) {
	suppressColor := NoColor()
	if suppressColor {
		// To avoid string templates with _and_ without colors, just overwrite color values
		// when no color will print out.
		red = ""
		cyan = ""
		reset = ""
		yellow = ""
		redBold = ""
		magenta = ""
	} else {
		red = RED
		cyan = CYAN
		reset = RESET
		yellow = YELLOW
		redBold = REDBOLD
		magenta = MAGENTA
	}
	if len(values) > 1 {
		var (
			sql             string
			formattedValues []string
			level           = values[0]
			currentTime     = fmt.Sprintf("\n%s[%s]%s", yellow, NowFunc().Format("2006-01-02 15:04:05"), reset)
			source          = fmt.Sprintf("%s(%v)%s", magenta, values[1], reset)
		)

		messages = []interface{}{source, currentTime}

		if len(values) == 2 {
			//remove the line break
			currentTime = currentTime[1:]
			//remove the brackets
			source = fmt.Sprintf("%s%v%s", magenta, values[1], reset)

			messages = []interface{}{currentTime, source}
		}

		if level == "sql" {
			// duration
			messages = append(messages, fmt.Sprintf(" %s[%.2fms]%s ", cyan, float64(values[2].(time.Duration).Nanoseconds()/1e4)/100.0, reset))

			// sql
			for _, value := range values[4].([]interface{}) {
				indirectValue := reflect.Indirect(reflect.ValueOf(value))
				if indirectValue.IsValid() {
					value = indirectValue.Interface()
					if t, ok := value.(time.Time); ok {
						if t.IsZero() {
							formattedValues = append(formattedValues, fmt.Sprintf("'%v'", "0000-00-00 00:00:00"))
						} else {
							formattedValues = append(formattedValues, fmt.Sprintf("'%v'", t.Format("2006-01-02 15:04:05")))
						}
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
						switch value.(type) {
						case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
							formattedValues = append(formattedValues, fmt.Sprintf("%v", value))
						default:
							formattedValues = append(formattedValues, fmt.Sprintf("'%v'", value))
						}
					}
				} else {
					formattedValues = append(formattedValues, "NULL")
				}
			}

			// differentiate between $n placeholders or else treat like ?
			if numericPlaceHolderRegexp.MatchString(values[3].(string)) {
				sql = values[3].(string)
				for index, value := range formattedValues {
					placeholder := fmt.Sprintf(`\$%d([^\d]|$)`, index+1)
					sql = regexp.MustCompile(placeholder).ReplaceAllString(sql, value+"$1")
				}
			} else {
				formattedValuesLength := len(formattedValues)
				for index, value := range sqlRegexp.Split(values[3].(string), -1) {
					sql += value
					if index < formattedValuesLength {
						sql += formattedValues[index]
					}
				}
			}

			messages = append(messages, sql)
			messages = append(messages, fmt.Sprintf(" \n%s[%s rows affected or returned]%s ", red, strconv.FormatInt(values[5].(int64), 10), reset))
		} else {
			messages = append(messages, redBold)
			messages = append(messages, values[2:]...)
			messages = append(messages, reset)
		}
	}

	return
}

type logger interface {
	Print(v ...interface{})
}

// LogWriter log writer interface
type LogWriter interface {
	Println(v ...interface{})
}

// Logger default logger
type Logger struct {
	LogWriter
}

// Print format & print log
func (logger Logger) Print(values ...interface{}) {
	logger.Println(LogFormatter(values...)...)
}

type nopLogger struct{}

func (nopLogger) Print(values ...interface{}) {}
