package gorm

import (
	"fmt"
	"log"
	"os"
)

var logger interface{}
var logger_disabled bool

type Logger interface {
	Print(v ...interface{})
}

func Print(level string, v ...interface{}) {
	if logger_disabled {
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
	go Print("warn", v...)
}

func info(v ...interface{}) {
	go Print("info", v...)
}

func debug(v ...interface{}) {
	go Print("debug", v...)
}
