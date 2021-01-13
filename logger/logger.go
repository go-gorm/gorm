package logger

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"gorm.io/gorm/utils"
)

// Colors
const (
	Reset       = "\033[0m"
	Red         = "\033[31m"
	Green       = "\033[32m"
	Yellow      = "\033[33m"
	Blue        = "\033[34m"
	Magenta     = "\033[35m"
	Cyan        = "\033[36m"
	White       = "\033[37m"
	BlueBold    = "\033[34;1m"
	MagentaBold = "\033[35;1m"
	RedBold     = "\033[31;1m"
	YellowBold  = "\033[33;1m"
)

// LogLevel
type LogLevel int

const (
	Silent LogLevel = iota + 1
	Error
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
	Info(context.Context, string, ...interface{})
	Warn(context.Context, string, ...interface{})
	Error(context.Context, string, ...interface{})
	Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error)
}

var (
	Discard = New(log.New(ioutil.Discard, "", log.LstdFlags), Config{})
	Default = New(log.New(os.Stdout, "\r\n", log.LstdFlags), Config{
		SlowThreshold: 200 * time.Millisecond,
		LogLevel:      Warn,
		Colorful:      true,
	})
	Recorder = traceRecorder{Interface: Default, BeginAt: time.Now()}
)

func New(writer Writer, config Config) Interface {
	var (
		infoStr      = "%s\n[info] "
		warnStr      = "%s\n[warn] "
		errStr       = "%s\n[error] "
		traceStr     = "%s\n[%.3fms] [rows:%v] %s"
		traceWarnStr = "%s %s\n[%.3fms] [rows:%v] %s"
		traceErrStr  = "%s %s\n[%.3fms] [rows:%v] %s"
	)

	if config.Colorful {
		infoStr = Green + "%s\n" + Reset + Green + "[info] " + Reset
		warnStr = BlueBold + "%s\n" + Reset + Magenta + "[warn] " + Reset
		errStr = Magenta + "%s\n" + Reset + Red + "[error] " + Reset
		traceStr = Green + "%s\n" + Reset + Yellow + "[%.3fms] " + BlueBold + "[rows:%v]" + Reset + " %s"
		traceWarnStr = Green + "%s " + Yellow + "%s\n" + Reset + RedBold + "[%.3fms] " + Yellow + "[rows:%v]" + Magenta + " %s" + Reset
		traceErrStr = RedBold + "%s " + MagentaBold + "%s\n" + Reset + Yellow + "[%.3fms] " + BlueBold + "[rows:%v]" + Reset + " %s"
	}

	return &logger{
		Writer:       writer,
		Config:       config,
		infoStr:      infoStr,
		warnStr:      warnStr,
		errStr:       errStr,
		traceStr:     traceStr,
		traceWarnStr: traceWarnStr,
		traceErrStr:  traceErrStr,
	}
}

type logger struct {
	Writer
	Config
	infoStr, warnStr, errStr            string
	traceStr, traceErrStr, traceWarnStr string
}

func (l *logger) printf(ctx context.Context, str, msg string, level LogLevel, data ...interface{}) {
	if l.LogLevel >= level {
		l.Printf(str+msg, append([]interface{}{utils.FileWithLineNum()}, data...)...)
	}
}

func (l *logger) trace(ctx context.Context, fc func() (string, int64), ft string, used time.Duration, trace interface{}) {
	sql, rows := fc()
	if rows == -1 {
		l.Printf(ft, utils.FileWithLineNum(), trace, float64(used.Nanoseconds())/1e6, "-", sql)
	} else {
		l.Printf(ft, utils.FileWithLineNum(), trace, float64(used.Nanoseconds())/1e6, rows, sql)
	}
}

// LogMode log mode
func (l *logger) LogMode(level LogLevel) Interface {
	newlogger := *l
	newlogger.LogLevel = level
	return &newlogger
}

// Info print info
func (l logger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.printf(ctx, l.infoStr, msg, Info, data...)
}

// Warn print warn messages
func (l logger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.printf(ctx, l.warnStr, msg, Warn, data...)
}

// Error print error messages
func (l logger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.printf(ctx, l.errStr, msg, Error, data...)
}

// Trace print sql message
func (l logger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel > Silent {
		elapsed := time.Since(begin)
		switch {
		case l.LogLevel == Info:
			l.trace(ctx, fc, l.traceStr, elapsed, err)
		case err != nil && l.LogLevel >= Error:
			l.trace(ctx, fc, l.traceErrStr, elapsed, err)
		case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= Warn:
			slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
			l.trace(ctx, fc, l.traceWarnStr, elapsed, slowLog)
		}
	}
}

type traceRecorder struct {
	Interface
	BeginAt      time.Time
	SQL          string
	RowsAffected int64
	Err          error
}

func (l traceRecorder) New() *traceRecorder {
	return &traceRecorder{Interface: l.Interface, BeginAt: time.Now()}
}

func (l *traceRecorder) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	l.BeginAt = begin
	l.SQL, l.RowsAffected = fc()
	l.Err = err
}
