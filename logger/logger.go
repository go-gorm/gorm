package logger

import (
	"fmt"
	"log"
	"os"
)

type LogLevel int

var Default Interface = Logger{Writer: log.New(os.Stdout, "\r\n", log.LstdFlags)}

const (
	Info LogLevel = iota + 1
	Warn
	Error
)

// Interface logger interface
type Interface interface {
	LogMode(LogLevel) Interface
	Info(string, ...interface{})
	Warn(string, ...interface{})
	Error(string, ...interface{})
}

// Writer log writer interface
type Writer interface {
	Print(...interface{})
}

type Logger struct {
	Writer
	logLevel LogLevel
}

func (logger Logger) LogMode(level LogLevel) Interface {
	return Logger{Writer: logger.Writer, logLevel: level}
}

// Info print info
func (logger Logger) Info(msg string, data ...interface{}) {
	if logger.logLevel <= Info {
		logger.Print("[info] " + fmt.Sprintf(msg, data...))
	}
}

// Warn print warn messages
func (logger Logger) Warn(msg string, data ...interface{}) {
	if logger.logLevel <= Warn {
		logger.Print("[warn] " + fmt.Sprintf(msg, data...))
	}
}

// Error print error messages
func (logger Logger) Error(msg string, data ...interface{}) {
	if logger.logLevel <= Error {
		logger.Print("[error] " + fmt.Sprintf(msg, data...))
	}
}
