package logger

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"gorm.io/gorm/utils"
)

// ZerologLogger implements Interface using zerolog
type ZerologLogger struct {
	Logger                    zerolog.Logger
	LogLevel                  LogLevel
	SlowThreshold             time.Duration
	Parameterized             bool
	IgnoreRecordNotFoundError bool
}

// NewZerologLogger creates a new logger using zerolog
func NewZerologLogger(logger zerolog.Logger, config Config) Interface {
	return &ZerologLogger{
		Logger:                    logger,
		LogLevel:                  config.LogLevel,
		SlowThreshold:             config.SlowThreshold,
		Parameterized:             config.ParameterizedQueries,
		IgnoreRecordNotFoundError: config.IgnoreRecordNotFoundError,
	}
}

// NewZerologLoggerWithConfig creates a new zerolog logger with custom configuration
func NewZerologLoggerWithConfig(config Config, output ...zerolog.Context) Interface {
	var logger zerolog.Logger

	if len(output) > 0 {
		logger = output[0].Logger()
	} else {
		// Default configuration
		consoleWriter := zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
			w.Out = os.Stdout
			w.TimeFormat = time.RFC3339
		})
		logger = zerolog.New(consoleWriter).
			Level(ZerologLevel(config.LogLevel)).
			With().
			Timestamp().
			Logger()
	}

	return NewZerologLogger(logger, config)
}

// LogMode sets the log level
func (l *ZerologLogger) LogMode(level LogLevel) Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

// Info logs info messages
func (l *ZerologLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Info {
		logger := l.Logger.Info().
			Str("file", utils.FileWithLineNum()).
			Interface("data", data)

		if ctx != nil {
			logger = logger.Ctx(ctx)
		}

		logger.Msg(msg)
	}
}

// Warn logs warning messages
func (l *ZerologLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Warn {
		logger := l.Logger.Warn().
			Str("file", utils.FileWithLineNum()).
			Interface("data", data)

		if ctx != nil {
			logger = logger.Ctx(ctx)
		}

		logger.Msg(msg)
	}
}

// Error logs error messages
func (l *ZerologLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= Error {
		logger := l.Logger.Error().
			Str("file", utils.FileWithLineNum()).
			Interface("data", data)

		if ctx != nil {
			logger = logger.Ctx(ctx)
		}

		logger.Msg(msg)
	}
}

// Trace logs SQL execution details
func (l *ZerologLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel <= Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	// Create base event
	var event *zerolog.Event

	switch {
	case err != nil && (!l.IgnoreRecordNotFoundError || !errors.Is(err, ErrRecordNotFound)):
		event = l.Logger.Error().Err(err)
	case l.SlowThreshold != 0 && elapsed > l.SlowThreshold:
		event = l.Logger.Warn().
			Str("slow_threshold", l.SlowThreshold.String())
	case l.LogLevel >= Info:
		event = l.Logger.Info()
	default:
		return
	}

	// Add common fields
	event = event.
		Str("file", utils.FileWithLineNum()).
		Str("duration", fmt.Sprintf("%.3fms", float64(elapsed.Nanoseconds())/1e6)).
		Str("sql", sql)

	if rows != -1 {
		event = event.Int64("rows", rows)
	}

	if ctx != nil {
		event = event.Ctx(ctx)
	}

	event.Msg("SQL executed")
}

// ParamsFilter filters SQL parameters
func (l *ZerologLogger) ParamsFilter(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
	if l.Parameterized {
		return sql, nil
	}
	return sql, params
}

// WithContext returns a logger with context
func (l *ZerologLogger) WithContext(ctx context.Context) *ZerologLogger {
	if ctx == nil {
		return l
	}

	newLogger := *l
	newLogger.Logger = l.Logger.With().Ctx(ctx).Logger()
	return &newLogger
}

// With adds fields to the logger
func (l *ZerologLogger) With() zerolog.Context {
	return l.Logger.With()
}

// Level creates a child logger with the minimum level set
func (l *ZerologLogger) Level(level zerolog.Level) zerolog.Logger {
	return l.Logger.Level(level)
}

// Sample returns a logger with sampling enabled
func (l *ZerologLogger) Sample(s zerolog.Sampler) zerolog.Logger {
	return l.Logger.Sample(s)
}

// Hook returns a logger with the specified hook
func (l *ZerologLogger) Hook(h zerolog.Hook) zerolog.Logger {
	return l.Logger.Hook(h)
}

// ZerologLevel converts LogLevel to zerolog.Level
func ZerologLevel(level LogLevel) zerolog.Level {
	switch level {
	case Silent:
		return zerolog.NoLevel
	case Error:
		return zerolog.ErrorLevel
	case Warn:
		return zerolog.WarnLevel
	case Info:
		return zerolog.InfoLevel
	default:
		return zerolog.InfoLevel
	}
}
