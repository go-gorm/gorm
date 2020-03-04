package logger

import (
	"log"
	"os"
	"time"

	"github.com/jinzhu/gorm/utils"
)

// Colors
const (
	Reset      = "\033[0m"
	Red        = "\033[31m"
	Green      = "\033[32m"
	Yellow     = "\033[33m"
	Blue       = "\033[34m"
	Magenta    = "\033[35m"
	Cyan       = "\033[36m"
	White      = "\033[37m"
	Redbold    = "\033[31;1m"
	YellowBold = "\033[33;1m"
)

// LogLevel
type LogLevel int

const (
	Error LogLevel = iota + 1
	Warn
	Info
)

// Writer log writer interface
type Writer interface {
	Printf(string, ...interface{})
}

type Config struct {
	SlowThreshold time.Duration
	Colorful      bool
	LogLevel      LogLevel
}

// Interface logger interface
type Interface interface {
	LogMode(LogLevel) Interface
	Info(string, ...interface{})
	Warn(string, ...interface{})
	Error(string, ...interface{})
	Trace(begin time.Time, fc func() (string, int64), err error)
}

var Default = New(log.New(os.Stdout, "\r\n", log.LstdFlags), Config{
	SlowThreshold: 100 * time.Millisecond,
	LogLevel:      Warn,
	Colorful:      true,
})

func New(writer Writer, config Config) Interface {
	var (
		infoPrefix     = "%s\n[info] "
		warnPrefix     = "%s\n[warn] "
		errPrefix      = "%s\n[error] "
		tracePrefix    = "%s\n[%v] [rows:%d] %s"
		traceErrPrefix = "%s\n[%v] [rows:%d] %s"
	)

	if config.Colorful {
		infoPrefix = Green + "%s\n" + Reset + Green + "[info] " + Reset
		warnPrefix = Blue + "%s\n" + Reset + Magenta + "[warn] " + Reset
		errPrefix = Magenta + "%s\n" + Reset + Red + "[error] " + Reset
		tracePrefix = Green + "%s\n" + Reset + YellowBold + "[%.3fms] " + Green + "[rows:%d]" + Reset + " %s"
		traceErrPrefix = Magenta + "%s\n" + Reset + Redbold + "[%.3fms] " + Yellow + "[rows:%d]" + Reset + " %s"
	}

	return logger{
		Writer:         writer,
		Config:         config,
		infoPrefix:     infoPrefix,
		warnPrefix:     warnPrefix,
		errPrefix:      errPrefix,
		tracePrefix:    tracePrefix,
		traceErrPrefix: traceErrPrefix,
	}
}

type logger struct {
	Writer
	Config
	infoPrefix, warnPrefix, errPrefix string
	tracePrefix, traceErrPrefix       string
}

// LogMode log mode
func (l logger) LogMode(level LogLevel) Interface {
	l.LogLevel = level
	return l
}

// Info print info
func (l logger) Info(msg string, data ...interface{}) {
	if l.LogLevel >= Info {
		l.Printf(l.infoPrefix+msg, append([]interface{}{utils.FileWithLineNum()}, data...)...)
	}
}

// Warn print warn messages
func (l logger) Warn(msg string, data ...interface{}) {
	if l.LogLevel >= Warn {
		l.Printf(l.warnPrefix+msg, append([]interface{}{utils.FileWithLineNum()}, data...)...)
	}
}

// Error print error messages
func (l logger) Error(msg string, data ...interface{}) {
	if l.LogLevel >= Error {
		l.Printf(l.errPrefix+msg, append([]interface{}{utils.FileWithLineNum()}, data...)...)
	}
}

// Trace print sql message
func (l logger) Trace(begin time.Time, fc func() (string, int64), err error) {
	if elapsed := time.Now().Sub(begin); err != nil || (elapsed > l.SlowThreshold && l.SlowThreshold != 0) {
		sql, rows := fc()
		fileline := utils.FileWithLineNum()
		if err != nil {
			fileline += " " + err.Error()
		}
		l.Printf(l.traceErrPrefix, fileline, float64(elapsed.Nanoseconds())/1e6, rows, sql)
	} else if l.LogLevel >= Info {
		sql, rows := fc()
		l.Printf(l.tracePrefix, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, rows, sql)
	}
}
