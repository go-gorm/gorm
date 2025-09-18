//go:build go1.21

package logger

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
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
		l.Logger.InfoContext(ctx, msg, slog.Any("data", data))
	}
}

func (l *slogLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Warn {
		l.Logger.WarnContext(ctx, msg, slog.Any("data", data))
	}
}

func (l *slogLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Error {
		l.Logger.ErrorContext(ctx, msg, slog.Any("data", data))
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
		l.Logger.ErrorContext(ctx, "SQL executed", slog.Attr{
			Key:   "trace",
			Value: slog.GroupValue(fields...),
		})

	case l.SlowThreshold != 0 && elapsed > l.SlowThreshold:
		l.Logger.WarnContext(ctx, "SQL executed", slog.Attr{
			Key:   "trace",
			Value: slog.GroupValue(fields...),
		})

	case l.LogLevel >= Info:
		l.Logger.InfoContext(ctx, "SQL executed", slog.Attr{
			Key:   "trace",
			Value: slog.GroupValue(fields...),
		})
	}
}

// ParamsFilter filter params
func (l *slogLogger) ParamsFilter(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if l.Parameterized {
		return sql, nil
	}
	return sql, params
}
