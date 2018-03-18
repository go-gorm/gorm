package logger

import (
	"fmt"
	"log"
	"os"
)

// Interface logger interface
type Interface interface {
	SQL(data ...interface{})
	Info(msg string, data ...interface{})
	Warn(msg string, data ...interface{})
	Error(msg string, data ...interface{})
}

// LogLevel log level
type LogLevel int

// DefaultLogLevel default log level
var DefaultLogLevel LogLevel

// DefaultLogger default logger
var DefaultLogger = Logger{log.New(os.Stdout, "\r\n", 0)}

const (
	// Info print SQL, warn messages and errors
	Info LogLevel = iota + 1
	// Warn print warn messages and errors
	Warn
	// Error print errors
	Error
)

func init() {
	switch os.Getenv("GORM_LOG_LEVEL") {
	case "info":
		DefaultLogLevel = Info
	case "warn":
		DefaultLogLevel = Warn
	case "error":
		DefaultLogLevel = Error
	default:
		DefaultLogLevel = Error
	}
}

// LogWriter log writer interface
type LogWriter interface {
	Println(v ...interface{})
}

// Logger logger
type Logger struct {
	LogWriter
}

// SQL print SQL statements
func (logger Logger) SQL(data ...interface{}) {
}

// Info print info
func (logger Logger) Info(message string, data ...interface{}) {
	// TODO show file line number
	logger.Println(fmt.Sprintf("[info] %v", fmt.Sprintf(message, data...)))
}

// Warn print warn messages
func (logger Logger) Warn(message string, data ...interface{}) {
	// TODO show file line number
	logger.Println(fmt.Sprintf("[warn] %v", fmt.Sprintf(message, data...)))
}

// Error print error messages
func (logger Logger) Error(message string, data ...interface{}) {
	// TODO show file line number
	logger.Println(fmt.Sprintf("[error] %v", fmt.Sprintf(message, data...)))
}
