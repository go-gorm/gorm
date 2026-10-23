//go:build go1.21

package logger

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm/utils"
)

// slogLogger adapts GORM diagnostics to a shared slog.Logger. LogMode copies adapter
// settings; the underlying slog handler determines final level enablement and output.
type slogLogger struct {
	// Logger is the shared structured logger; its handler must accept concurrent calls.
	Logger *slog.Logger
	// LogLevel gates ordinary GORM messages and Silent/Info tracing. Error and slow trace
	// branches also depend on the slog handler's Enabled result.
	LogLevel LogLevel
	// SlowThreshold classifies elapsed SQL time as a warning; zero disables that branch.
	SlowThreshold time.Duration
	// Parameterized suppresses bound parameters during GORM trace formatting, leaving SQL
	// literals unchanged.
	Parameterized bool
	// Colorful is unused by this adapter; formatting belongs to the slog handler.
	Colorful bool // Ignored in slog
	// IgnoreRecordNotFoundError excludes the not-found sentinel from the trace error branch
	// without changing the operation result.
	IgnoreRecordNotFoundError bool
}

// NewSlogLogger adapts a non-nil slog.Logger using config. Available with Go 1.21 or
// later, it preserves structured trace fields and external caller information.
// Info/Warn/Error data is stored as a structured data attribute rather than printf
// interpolation. The handler's Enabled policy applies in addition to adapter settings;
// Colorful is ignored.
func NewSlogLogger(logger *slog.Logger, config Config) Interface {
	return &slogLogger{
		Logger:                    logger,
		LogLevel:                  config.LogLevel,
		SlowThreshold:             config.SlowThreshold,
		Parameterized:             config.ParameterizedQueries,
		IgnoreRecordNotFoundError: config.IgnoreRecordNotFoundError,
	}
}

// LogMode returns a shallow adapter copy with the requested GORM level, sharing the
// same slog.Logger.
func (l *slogLogger) LogMode(level LogLevel) Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info emits a structured info message when the GORM level permits it. data is an
// attribute slice, not substituted into msg as printf arguments.
func (l *slogLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Info {
		l.log(ctx, slog.LevelInfo, msg, slog.Any("data", data))
	}
}

// Warn emits a structured warning when the GORM level permits it and the slog handler
// enables that level.
func (l *slogLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Warn {
		l.log(ctx, slog.LevelWarn, msg, slog.Any("data", data))
	}
}

// Error emits a structured error message with accompanying data, without altering
// DB.Error.
func (l *slogLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Error {
		l.log(ctx, slog.LevelError, msg, slog.Any("data", data))
	}
}

// Trace evaluates fc unless Silent, then selects an error, slow warning, or Info trace.
// Error/slow branches are filtered by the slog handler; they do not use the printf
// logger's identical GORM level checks. Rows=-1 omits the rows attribute. SQL is a
// diagnostic representation.
func (l *slogLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()
	fields := []slog.Attr{
		slog.String("duration", fmt.Sprintf("%.3fms", float64(elapsed.Nanoseconds())/1e6)),
		slog.String("sql", sql),
	}

	if rows != -1 {
		fields = append(fields, slog.Int64("rows", rows))
	}

	switch {
	case err != nil && (!l.IgnoreRecordNotFoundError || !errors.Is(err, ErrRecordNotFound)):
		fields = append(fields, slog.String("error", err.Error()))
		l.log(ctx, slog.LevelError, "SQL executed", slog.Attr{
			Key:   "trace",
			Value: slog.GroupValue(fields...),
		})

	case l.SlowThreshold != 0 && elapsed > l.SlowThreshold:
		l.log(ctx, slog.LevelWarn, "SQL executed", slog.Attr{
			Key:   "trace",
			Value: slog.GroupValue(fields...),
		})

	case l.LogLevel >= Info:
		l.log(ctx, slog.LevelInfo, "SQL executed", slog.Attr{
			Key:   "trace",
			Value: slog.GroupValue(fields...),
		})
	}
}

// log substitutes context.Background for nil context, checks handler enablement, and
// emits a record with the first relevant external caller PC. Handler errors are ignored
// because the logging interface has no error return.
func (l *slogLogger) log(ctx context.Context, level slog.Level, msg string, args ...any) {
	if ctx == nil {
		ctx = context.Background()
	}

	if !l.Logger.Enabled(ctx, level) {
		return
	}

	r := slog.NewRecord(time.Now(), level, msg, utils.CallerFrame().PC)
	r.Add(args...)
	_ = l.Logger.Handler().Handle(ctx, r)
}

// ParamsFilter omits bound arguments when Parameterized is true. It does not remove
// literals from SQL or redact independent structured diagnostic data.
func (l *slogLogger) ParamsFilter(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if l.Parameterized {
		return sql, nil
	}
	return sql, params
}
