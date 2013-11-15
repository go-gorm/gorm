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

func (s *Chain) print(level string, v ...interface{}) {
	if s.d.logMode || s.debug_mode || level == "debug" {
		if _, ok := s.d.logger.(Logger); !ok {
			fmt.Println("logger haven't been set, using os.Stdout")
			s.d.logger = default_logger
		}
		args := []interface{}{level}
		s.d.logger.(Logger).Print(append(args, v...)...)
	}
}

func (s *Chain) warn(v ...interface{}) {
	go s.print("warn", v...)
}

func (s *Chain) info(v ...interface{}) {
	go s.print("info", v...)
}

func (s *Chain) slog(sql string, t time.Time, vars ...interface{}) {
	go s.print("sql", time.Now().Sub(t), sql, vars)
}

func (s *Chain) debug(v ...interface{}) {
	go s.print("debug", v...)
}
