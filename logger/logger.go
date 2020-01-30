package logger

type LogLevel int

const (
	Info LogLevel = iota + 1
	Warn
	Error
)

// Interface logger interface
type Interface interface {
	LogMode(LogLevel) Interface
}
