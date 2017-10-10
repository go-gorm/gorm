package logger

import "os"

// Interface logger interface
type Interface interface {
}

// LogLevel log level
type LogLevel int

// DefaultLogLevel default log level
var DefaultLogLevel LogLevel

const (
	// Info print SQL, warn messages and errors
	Info LogLevel = 1 << iota
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
