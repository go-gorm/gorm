package logger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm/utils"
)

// LogrusLogger implements Interface using logrus
type LogrusLogger struct {
	Logger                    *logrus.Logger
	LogLevel                  LogLevel
	SlowThreshold             time.Duration
	Parameterized             bool
	IgnoreRecordNotFoundError bool
}

// NewLogrusLogger creates a new logger using logrus
func NewLogrusLogger(logger *logrus.Logger, config Config) Interface {
	return &LogrusLogger{
		Logger:                    logger,
		LogLevel:                  config.LogLevel,
		SlowThreshold:             config.SlowThreshold,
		Parameterized:             config.ParameterizedQueries,
		IgnoreRecordNotFoundError: config.IgnoreRecordNotFoundError,
	}
}

// LogMode sets the log level
func (l *LogrusLogger) LogMode(level LogLevel) Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info logs info messages
func (l *LogrusLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Info {
		fields := logrus.Fields{
			"file": utils.FileWithLineNum(),
			"data": data,
		}
		l.Logger.WithContext(ctx).WithFields(fields).Info(msg)
	}
}

// Warn logs warning messages
func (l *LogrusLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Warn {
		fields := logrus.Fields{
			"file": utils.FileWithLineNum(),
			"data": data,
		}
		l.Logger.WithContext(ctx).WithFields(fields).Warn(msg)
	}
}

// Error logs error messages
func (l *LogrusLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Error {
		fields := logrus.Fields{
			"file": utils.FileWithLineNum(),
			"data": data,
		}
		l.Logger.WithFields(fields).Error(msg)
	}
}

// Trace logs SQL execution details
func (l *LogrusLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := logrus.Fields{
		"file":     utils.FileWithLineNum(),
		"duration": fmt.Sprintf("%.3fms", float64(elapsed.Nanoseconds())/1e6),
		"sql":      sql,
	}

	if rows != -1 {
		fields["rows"] = rows
	}

	logger := l.Logger.WithContext(ctx).WithFields(fields)

	switch {
	case err != nil && (!l.IgnoreRecordNotFoundError || !errors.Is(err, ErrRecordNotFound)):
		fields["error"] = err.Error()
		logger.WithFields(fields).Error("SQL executed")

	case l.SlowThreshold != 0 && elapsed > l.SlowThreshold:
		fields["slow_threshold"] = l.SlowThreshold.String()
		logger.WithFields(fields).Warn("SLOW SQL executed")

	case l.LogLevel >= Info:
		logger.WithFields(fields).Info("SQL executed")
	}
}

// ParamsFilter filters SQL parameters
func (l *LogrusLogger) ParamsFilter(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if l.Parameterized {
		return sql, nil
	}
	return sql, params
}

// WithContext returns a logger with context
func (l *LogrusLogger) WithContext(ctx context.Context) *LogrusLogger {
	if ctx == nil {
		return l
	}

	newLogger := *l
	newLogger.Logger = l.Logger.WithContext(ctx).Logger
	return &newLogger
}

// WithField adds a field to the logger
func (l *LogrusLogger) WithField(key string, value interface{}) *LogrusLogger {
	newLogger := *l
	newLogger.Logger = l.Logger.WithField(key, value).Logger
	return &newLogger
}

// WithFields adds multiple fields to the logger
func (l *LogrusLogger) WithFields(fields logrus.Fields) *LogrusLogger {
	newLogger := *l
	newLogger.Logger = l.Logger.WithFields(fields).Logger
	return &newLogger
}
