package logger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm/utils"
)

// ZapLogger implements Interface using zap
type ZapLogger struct {
	Logger                    *zap.Logger
	LogLevel                  LogLevel
	SlowThreshold             time.Duration
	Parameterized             bool
	IgnoreRecordNotFoundError bool
}

// NewZapLogger creates a new logger using zap
func NewZapLogger(logger *zap.Logger, config Config) Interface {
	return &ZapLogger{
		Logger:                    logger,
		LogLevel:                  config.LogLevel,
		SlowThreshold:             config.SlowThreshold,
		Parameterized:             config.ParameterizedQueries,
		IgnoreRecordNotFoundError: config.IgnoreRecordNotFoundError,
	}
}

// NewZapLoggerWithConfig creates a new zap logger with custom configuration
func NewZapLoggerWithConfig(config Config, zapConfig ...zap.Config) Interface {
	var zapCfg zap.Config
	if len(zapConfig) > 0 {
		zapCfg = zapConfig[0]
	} else {
		// Default production configuration
		zapCfg = zap.NewProductionConfig()
		zapCfg.Level = zap.NewAtomicLevelAt(ZapLevel(config.LogLevel))
	}

	logger, err := zapCfg.Build()
	if err != nil {
		// Fallback to development config
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.Level = zap.NewAtomicLevelAt(ZapLevel(config.LogLevel))
		logger, _ = zapCfg.Build()
	}

	return NewZapLogger(logger, config)
}

// LogMode sets the log level
func (l *ZapLogger) LogMode(level LogLevel) Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info logs info messages
func (l *ZapLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Info {
		fields := []zap.Field{
			zap.String("file", utils.FileWithLineNum()),
			zap.Any("data", data),
		}
		l.Logger.Info(msg, fields...)
	}
}

// Warn logs warning messages
func (l *ZapLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Warn {
		fields := []zap.Field{
			zap.String("file", utils.FileWithLineNum()),
			zap.Any("data", data),
		}
		l.Logger.Warn(msg, fields...)
	}
}

// Error logs error messages
func (l *ZapLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Error {
		fields := []zap.Field{
			zap.String("file", utils.FileWithLineNum()),
			zap.Any("data", data),
		}
		l.Logger.Error(msg, fields...)
	}
}

// Trace logs SQL execution details
func (l *ZapLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := []zap.Field{
		zap.String("file", utils.FileWithLineNum()),
		zap.String("duration", fmt.Sprintf("%.3fms", float64(elapsed.Nanoseconds())/1e6)),
		zap.String("sql", sql),
	}

	if rows != -1 {
		fields = append(fields, zap.Int64("rows", rows))
	}

	switch {
	case err != nil && (!l.IgnoreRecordNotFoundError || !errors.Is(err, ErrRecordNotFound)):
		fields = append(fields, zap.Error(err))
		l.Logger.Error("SQL executed", fields...)

	case l.SlowThreshold != 0 && elapsed > l.SlowThreshold:
		fields = append(fields, zap.String("slow_threshold", l.SlowThreshold.String()))
		l.Logger.Warn("SLOW SQL executed", fields...)

	case l.LogLevel >= Info:
		l.Logger.Info("SQL executed", fields...)
	}
}

// ParamsFilter filters SQL parameters
func (l *ZapLogger) ParamsFilter(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if l.Parameterized {
		return sql, nil
	}
	return sql, params
}

// WithContext returns a logger with context
func (l *ZapLogger) WithContext(ctx context.Context) *ZapLogger {
	if ctx == nil {
		return l
	}

	newLogger := *l
	// Zap doesn't have direct context support, but we can create a new logger with context
	newLogger.Logger = l.Logger.With(zap.Any("context", ctx))
	return &newLogger
}

// WithField adds a field to the logger
func (l *ZapLogger) WithField(key string, value interface{}) *ZapLogger {
	newLogger := *l
	newLogger.Logger = l.Logger.With(zap.Any(key, value))
	return &newLogger
}

// WithFields adds multiple fields to the logger
func (l *ZapLogger) WithFields(fields ...zap.Field) *ZapLogger {
	newLogger := *l
	newLogger.Logger = l.Logger.With(fields...)
	return &newLogger
}

// Sugar returns a sugared logger
func (l *ZapLogger) Sugar() *zap.SugaredLogger {
	return l.Logger.Sugar()
}

// ZapLevel converts LogLevel to zapcore.Level
func ZapLevel(level LogLevel) zapcore.Level {
	switch level {
	case Silent:
		return zapcore.DPanicLevel // Use DPanic for silent to avoid actual logging
	case Error:
		return zapcore.ErrorLevel
	case Warn:
		return zapcore.WarnLevel
	case Info:
		return zapcore.InfoLevel
	default:
		return zapcore.InfoLevel
	}
}
