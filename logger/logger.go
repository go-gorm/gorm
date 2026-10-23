package logger

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"gorm.io/gorm/utils"
)

// ErrRecordNotFound is the shared not-found sentinel also exported by gorm. Loggers may
// suppress it when IgnoreRecordNotFoundError is set; suppression changes logging, not
// the operation error.
var ErrRecordNotFound = errors.New("record not found")

// Colors
const (
	// Reset is the ANSI terminal escape sequence for reset formatting. It is formatting
	// text, not a logging level.
	Reset = "\033[0m"
	// Red is the ANSI terminal escape sequence for red. It is formatting text, not a
	// logging level.
	Red = "\033[31m"
	// Green is the ANSI terminal escape sequence for green. It is formatting text, not a
	// logging level.
	Green = "\033[32m"
	// Yellow is the ANSI terminal escape sequence for yellow. It is formatting text, not a
	// logging level.
	Yellow = "\033[33m"
	// Blue is the ANSI terminal escape sequence for blue. It is formatting text, not a
	// logging level.
	Blue = "\033[34m"
	// Magenta is the ANSI terminal escape sequence for magenta. It is formatting text, not
	// a logging level.
	Magenta = "\033[35m"
	// Cyan is the ANSI terminal escape sequence for cyan. It is formatting text, not a
	// logging level.
	Cyan = "\033[36m"
	// White is the ANSI terminal escape sequence for white. It is formatting text, not a
	// logging level.
	White = "\033[37m"
	// BlueBold is the ANSI terminal escape sequence for bold blue. It is formatting text,
	// not a logging level.
	BlueBold = "\033[34;1m"
	// MagentaBold is the ANSI terminal escape sequence for bold magenta. It is formatting
	// text, not a logging level.
	MagentaBold = "\033[35;1m"
	// RedBold is the ANSI terminal escape sequence for bold red. It is formatting text, not
	// a logging level.
	RedBold = "\033[31;1m"
	// YellowBold is the ANSI terminal escape sequence for bold yellow. It is formatting
	// text, not a logging level.
	YellowBold = "\033[33;1m"
)

// LogLevel orders GORM verbosity from Silent through Error, Warn, and Info. It is
// distinct from log/slog levels; adapters map between the two.
type LogLevel int

const (
	// Silent disables operation logging through the standard logger and the slog adapter.
	Silent LogLevel = iota + 1
	// Error enables error-level messages in the standard logger.
	Error
	// Warn enables warnings and errors, including configured slow-query warnings.
	Warn
	// Info enables informational messages and ordinary SQL traces in addition to warnings
	// and errors.
	Info
)

// Writer is the formatted-output sink used by New. A standard log.Logger satisfies it.
// Supply a writer safe for the concurrent operations that share the logger.
type Writer interface {
	// Printf writes a formatted log message. The caller supplies printf-style arguments,
	// including source and timing metadata.
	Printf(string, ...interface{})
}

// Config controls built-in logger formatting, trace thresholds, and parameter
// visibility. New uses these values as provided rather than filling Default's settings.
// NewSlogLogger uses the relevant options but ignores terminal colors.
type Config struct {
	// SlowThreshold triggers a warning when elapsed query time exceeds it. Zero disables
	// slow-query classification; it is not a database timeout.
	SlowThreshold time.Duration
	// Colorful enables ANSI color escapes for the formatted logger; it is ignored by the
	// slog adapter.
	Colorful bool
	// IgnoreRecordNotFoundError suppresses classification of ErrRecordNotFound as a trace
	// error. The query may still be logged for slowness or at Info level.
	IgnoreRecordNotFoundError bool
	// ParameterizedQueries omits bound arguments when GORM formats trace SQL. Literal
	// values already embedded in SQL and independent log messages are not redacted.
	ParameterizedQueries bool
	// LogLevel controls GORM verbosity. For slog output the underlying handler's Enabled
	// policy also applies.
	LogLevel LogLevel
}

// Interface receives GORM messages and operation traces. The context is the operation
// context; implementations should support concurrent use when shared by a DB. Trace
// receives a lazy SQL/row-count function so unused formatting can be skipped.
type Interface interface {
	// LogMode returns a logger at the requested level. Built-in implementations return a
	// shallow copy so changing one session's level does not mutate another's logger.
	LogMode(LogLevel) Interface
	// Info records an informational diagnostic with the operation context and accompanying
	// data.
	Info(context.Context, string, ...interface{})
	// Warn records a warning with the operation context and accompanying data.
	Warn(context.Context, string, ...interface{})
	// Error records an error-level diagnostic. This does not set or replace the database
	// operation's Error.
	Error(context.Context, string, ...interface{})
	// Trace records one operation beginning at begin, with lazy diagnostic SQL and affected
	// rows from fc. A row count of -1 means unavailable. err classifies failure; logged SQL
	// is for inspection, not execution.
	Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error)
}

var (
	// Discard is a logger configured with an io.Discard writer and zero-value settings. Use
	// it to suppress log output, not to suppress database errors.
	Discard = New(log.New(io.Discard, "", log.LstdFlags), Config{})
	// Default is the stdout logger used by GORM when no logger is configured. It uses Warn
	// level, a 200ms slow threshold, and color output. Prefer a per-DB logger instead of
	// reassigning this shared variable during concurrent use.
	Default = New(log.New(os.Stdout, "\r\n", log.LstdFlags), Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  Warn,
		IgnoreRecordNotFoundError: false,
		Colorful:                  true,
	})
	// Recorder is the template for per-operation trace recorders used by DB.Scan. Call New
	// for an independent recorder rather than mutating this shared instance.
	Recorder = traceRecorder{Interface: Default, BeginAt: time.Now()}

	// RecorderParamsFilter optionally filters parameters for trace recorders. It defaults
	// to returning inputs unchanged. Configure it before concurrent use; this shared
	// function variable is not synchronized.
	RecorderParamsFilter = func(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
		return sql, params
	}
)

// New constructs a formatted logger using writer and config exactly as supplied. It
// does not apply Default's Warn level or slow threshold automatically. Assign the
// result to gorm.Config.Logger or a session Logger. The writer and referenced resources
// remain shared across LogMode copies.
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

// logger is the printf-based implementation of Interface. Its configuration is copied
// by LogMode, while the output writer remains shared.
type logger struct {
	Writer
	Config
	// infoStr, warnStr, and errStr are source-prefixed printf formats for diagnostic
	// levels, optionally containing ANSI colors.
	infoStr, warnStr, errStr string
	// traceStr, traceErrStr, and traceWarnStr format ordinary, failed, and slow SQL traces
	// with elapsed time and row counts.
	traceStr, traceErrStr, traceWarnStr string
}

// LogMode returns a shallow copy with level applied, preserving the shared output
// writer and formatting strings.
func (l *logger) LogMode(level LogLevel) Interface {
	newlogger := *l
	newlogger.LogLevel = level
	return &newlogger
}

// Info writes a source-prefixed printf diagnostic when the configured level is at least
// Info.
func (l *logger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Info {
		l.Printf(l.infoStr+msg, append([]interface{}{utils.FileWithLineNum()}, data...)...)
	}
}

// Warn writes a source-prefixed printf diagnostic when the configured level is at least
// Warn.
func (l *logger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Warn {
		l.Printf(l.warnStr+msg, append([]interface{}{utils.FileWithLineNum()}, data...)...)
	}
}

// Error writes a source-prefixed printf diagnostic when the configured level is at
// least Error. It does not modify DB error state.
func (l *logger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Error {
		l.Printf(l.errStr+msg, append([]interface{}{utils.FileWithLineNum()}, data...)...)
	}
}

// Trace chooses error, slow-query warning, or ordinary Info output in that order. It
// invokes fc only when a message will be formatted, and renders rows=-1 as unavailable.
// IgnoreRecordNotFoundError suppresses only the error branch. This is diagnostic SQL,
// not executable parameter binding.
//
//nolint:cyclop
func (l *logger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= Silent {
		return
	}

	elapsed := time.Since(begin)
	switch {
	case err != nil && l.LogLevel >= Error && (!errors.Is(err, ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		sql, rows := fc()
		if rows == -1 {
			l.Printf(l.traceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.Printf(l.traceErrStr, utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case elapsed > l.SlowThreshold && l.SlowThreshold != 0 && l.LogLevel >= Warn:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
		if rows == -1 {
			l.Printf(l.traceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.Printf(l.traceWarnStr, utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case l.LogLevel == Info:
		sql, rows := fc()
		if rows == -1 {
			l.Printf(l.traceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.Printf(l.traceStr, utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	}
}

// ParamsFilter returns no bound parameters when ParameterizedQueries is enabled;
// otherwise it preserves them. SQL text itself is unchanged, so embedded literals
// remain visible.
func (l *logger) ParamsFilter(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if l.Config.ParameterizedQueries {
		return sql, nil
	}
	return sql, params
}

// traceRecorder captures one operation's SQL, elapsed-time origin, row count, and error
// while delegating other messages to its embedded logger. It is mutable and must not be
// shared by simultaneous operations.
type traceRecorder struct {
	Interface
	// BeginAt is the start time supplied to the last Trace call, or creation time before
	// tracing.
	BeginAt time.Time
	// SQL is the last diagnostic statement returned by the trace callback.
	SQL string
	// RowsAffected is the last traced row count; -1 can denote unavailable metadata.
	RowsAffected int64
	// Err is the last traced operation error, independent of log filtering.
	Err error
}

// New creates an independent recorder with the same underlying logger and a fresh start
// time. Prior captured SQL, rows, and errors are not copied.
func (l *traceRecorder) New() *traceRecorder {
	return &traceRecorder{Interface: l.Interface, BeginAt: time.Now()}
}

// Trace immediately calls fc and replaces the recorder's captured values; it does not
// emit a trace itself.
func (l *traceRecorder) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	l.BeginAt = begin
	l.SQL, l.RowsAffected = fc()
	l.Err = err
}

// ParamsFilter delegates to RecorderParamsFilter when non-nil, otherwise returning SQL
// and parameters unchanged.
func (l *traceRecorder) ParamsFilter(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if RecorderParamsFilter == nil {
		return sql, params
	}
	return RecorderParamsFilter(ctx, sql, params...)
}
