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

type slogLogger struct {
	Logger                    *slog.Logger
	LogLevel                  LogLevel
	SlowThreshold             time.Duration
	Parameterized             bool
	Colorful                  bool // Ignored in slog
	IgnoreRecordNotFoundError bool
}

func NewSlogLogger(logger *slog.Logger, config Config) Interface {
	return &slogLogger{
		Logger:                    logger,
		LogLevel:                  config.LogLevel,
		SlowThreshold:             config.SlowThreshold,
		Parameterized:             config.ParameterizedQueries,
		IgnoreRecordNotFoundError: config.IgnoreRecordNotFoundError,
	}
}

func (l *slogLogger) LogMode(level LogLevel) Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

func (l *slogLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Info {
		l.log(ctx, slog.LevelInfo, msg, slog.Any("data", data))
	}
}

func (l *slogLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Warn {
		l.log(ctx, slog.LevelWarn, msg, slog.Any("data", data))
	}
}

func (l *slogLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Error {
		l.log(ctx, slog.LevelError, msg, slog.Any("data", data))
	}
}

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

// ParamsFilter filter params
func (l *slogLogger) ParamsFilter(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if l.Parameterized {
		return sql, nil
	}
	return sql, params
}
