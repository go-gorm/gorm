package gorm

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"time"
)

type Logger interface {
	Print(v ...interface{})
}

type defaultLogger struct {
	*log.Logger
}

func (logger defaultLogger) Print(v ...interface{}) {
	if len(v) > 1 {
		level := v[0]
		tim := "\033[33m[" + time.Now().Format("2006-01-02 15:04:05") + "]\033[0m"

		if level == "sql" {
			dur := v[1]
			sql := fmt.Sprintf(regexp.MustCompile(`(\$\d+)|\?`).ReplaceAllString(v[2].(string), "'%v'"), v[3].([]interface{})...)
			dur = fmt.Sprintf(" \033[36;1m[%.2fms]\033[0m ", float64(dur.(time.Duration).Nanoseconds()/1e4)/100.0)
			logger.Println(tim, dur, sql)
		} else {
			messages := []interface{}{"\033[31m"}
			messages = append(messages, v...)
			messages = append(messages, "\033[0m")
			logger.Println(messages...)
		}
	}
}

var default_logger defaultLogger

func init() {
	// default_logger = log.New(os.Stdout, "\r\n", 0)
	default_logger = defaultLogger{log.New(os.Stdout, "\r\n", 0)}
}
