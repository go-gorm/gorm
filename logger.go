package gorm

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"time"
)

type logger interface {
	Print(v ...interface{})
}

type Logger struct {
	*log.Logger
}

var defaultLogger = Logger{log.New(os.Stdout, "\r\n", 0)}

// Format log
var sqlRegexp = regexp.MustCompile(`(\$\d+)|\?`)

func (logger Logger) Print(values ...interface{}) {
	if len(values) > 1 {
		level := values[0]
		currentTime := "\n\033[33m[" + NowFunc().Format("2006-01-02 15:04:05") + "]\033[0m"
		source := fmt.Sprintf("\033[35m(%v)\033[0m", values[1])
		messages := []interface{}{source, currentTime}

		if level == "sql" {
			// duration
			messages = append(messages, fmt.Sprintf(" \033[36;1m[%.2fms]\033[0m ", float64(values[2].(time.Duration).Nanoseconds()/1e4)/100.0))
			// sql
			messages = append(messages, fmt.Sprintf(sqlRegexp.ReplaceAllString(values[3].(string), "'%v'"), values[4].([]interface{})...))
		} else {
			messages = append(messages, "\033[31;1m")
			messages = append(messages, values[2:]...)
			messages = append(messages, "\033[0m")
		}
		logger.Println(messages...)
	}
}
