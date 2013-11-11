package gorm

import "fmt"

var logger interface{}

type Logger interface {
	Print(v ...interface{})
}

func Print(level string, v ...interface{}) {
	args := []interface{}{level}

	if l, ok := logger.(Logger); ok {
		l.Print(append(args, v...))
	} else {
		fmt.Println("logger haven't been set,", append(args, v...))
	}
}

func warn(v ...interface{}) {
	Print("warn", v...)
}

func info(v ...interface{}) {
	Print("info", v...)
}

func debug(v ...interface{}) {
	Print("debug", v...)
}
