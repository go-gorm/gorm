package gorm

import (
	"fmt"
	"log"
	"os"
	"regexp"
)

var logger interface{}
var logger_disabled bool

type Logger interface {
	Print(v ...interface{})
}

func print(level string, v ...interface{}) {
	if logger_disabled && level != "debug" {
		return
	}

	var has_valid_logger bool
	if logger, has_valid_logger = logger.(Logger); !has_valid_logger {
		fmt.Println("logger haven't been set, using os.Stdout")
		logger = log.New(os.Stdout, "", 0)
	}

	args := []interface{}{level}
	logger.(Logger).Print(append(args, v...))
}

func warn(v ...interface{}) {
	go print("warn", v...)
}

func info(v ...interface{}) {
	go print("info", v...)
}

func slog(sql string, vars ...interface{}) {
	go print("sql", fmt.Sprintf(regexp.MustCompile(`\$\d|\?`).ReplaceAllString(sql, "'%v'"), vars...))
}

func debug(v ...interface{}) {
	go print("debug", v...)
}
