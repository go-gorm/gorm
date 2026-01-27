package logger

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestNewZapLogger(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&buf),
		zapcore.InfoLevel,
	)

	zapLogger := zap.New(core)

	config := Config{
		LogLevel:      Info,
		SlowThreshold: 100 * time.Millisecond,
		Colorful:      true,
	}

	zapAdapter := NewZapLogger(zapLogger, config)

	require.NotNil(t, zapAdapter)
	assert.Equal(t, Info, zapAdapter.(*ZapLogger).LogLevel)
	assert.Equal(t, 100*time.Millisecond, zapAdapter.(*ZapLogger).SlowThreshold)
}

func TestZapLogger_LogMode(t *testing.T) {
	zapLogger := zap.NewNop()

	logger := NewZapLogger(zapLogger, Config{
		LogLevel: Error,
	})

	// Test changing log mode
	infoLogger := logger.LogMode(Info)
	assert.Equal(t, Info, infoLogger.(*ZapLogger).LogLevel)

	// Test that original is not affected
	assert.Equal(t, Error, logger.(*ZapLogger).LogLevel)
}

func TestZapLogger_LogLevels(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&buf),
		zapcore.InfoLevel,
	)

	zapLogger := zap.New(core)
	logger := NewZapLogger(zapLogger, Config{
		LogLevel: Info,
	})

	tests := []struct {
		name   string
		level  LogLevel
		logMsg string
	}{
		{"Info level", Info, "Test info message"},
		{"Warn level", Warn, "Test warn message"},
		{"Error level", Error, "Test error message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			logger := logger.LogMode(tt.level)

			switch tt.level {
			case Info:
				logger.Info(ctx, tt.logMsg, "key", "value")
			case Warn:
				logger.Warn(ctx, tt.logMsg, "key", "value")
			case Error:
				logger.Error(ctx, tt.logMsg, "key", "value")
			}

			output := buf.String()
			assert.Contains(t, output, tt.logMsg)
			assert.Contains(t, output, "key")
			assert.Contains(t, output, "value")
		})
	}
}

func TestZapLogger_Trace(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&buf),
		zapcore.InfoLevel,
	)

	zapLogger := zap.New(core)
	logger := NewZapLogger(zapLogger, Config{
		LogLevel:      Info,
		SlowThreshold: 100 * time.Millisecond,
	})

	t.Run("Normal trace", func(t *testing.T) {
		buf.Reset()
		logger.Trace(ctx, time.Now(), func() (string, int64) {
			return "SELECT * FROM users WHERE id = ?", 5
		}, nil)

		output := buf.String()
		assert.Contains(t, output, "SELECT * FROM users WHERE id = ?")
		assert.Contains(t, output, "rows")
		assert.Contains(t, output, "5")
		assert.Contains(t, output, "duration")
	})

	t.Run("Slow query", func(t *testing.T) {
		buf.Reset()
		logger.Trace(ctx, time.Now().Add(-150*time.Millisecond), func() (string, int64) {
			return "SELECT * FROM large_table", 1000
		}, nil)

		output := buf.String()
		assert.Contains(t, output, "SELECT * FROM large_table")
		assert.Contains(t, output, "SLOW")
		assert.Contains(t, output, "slow_threshold")
	})

	t.Run("Error trace", func(t *testing.T) {
		buf.Reset()
		testError := assert.AnError
		logger.Trace(ctx, time.Now(), func() (string, int64) {
			return "SELECT * FROM non_existent_table", 0
		}, testError)

		output := buf.String()
		assert.Contains(t, output, "SELECT * FROM non_existent_table")
		assert.Contains(t, output, "error")
	})

	t.Run("Record not found error with ignore", func(t *testing.T) {
		buf.Reset()
		logger := logger.LogMode(Error)
		logger.(*ZapLogger).IgnoreRecordNotFoundError = true

		logger.Trace(ctx, time.Now(), func() (string, int64) {
			return "SELECT * FROM empty_table", 0
		}, ErrRecordNotFound)

		output := buf.String()
		// Should not log when record not found error is ignored
		if logger.(*ZapLogger).IgnoreRecordNotFoundError {
			assert.Empty(t, output)
		}
	})
}

func TestZapLogger_ParamsFilter(t *testing.T) {
	ctx := context.Background()
	zapLogger := zap.NewNop()

	tests := []struct {
		name          string
		parameterized bool
		sql           string
		params        []interface{}
		expectParam   bool
	}{
		{
			name:          "With parameters",
			parameterized: false,
			sql:           "SELECT * FROM users WHERE id = ?",
			params:        []interface{}{1},
			expectParam:   true,
		},
		{
			name:          "Parameterized queries",
			parameterized: true,
			sql:           "SELECT * FROM users WHERE id = ?",
			params:        []interface{}{1},
			expectParam:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewZapLogger(zapLogger, Config{
				ParameterizedQueries: tt.parameterized,
			})

			sql, params := logger.(*ZapLogger).ParamsFilter(ctx, tt.sql, tt.params...)

			assert.Equal(t, tt.sql, sql)

			if tt.expectParam {
				assert.Equal(t, tt.params, params)
				assert.Equal(t, 1, len(params))
			} else {
				assert.Nil(t, params)
			}
		})
	}
}

func TestZapLogger_WithMethods(t *testing.T) {
	zapLogger := zap.NewNop()

	logger := NewZapLogger(zapLogger, Config{
		LogLevel: Info,
	})

	t.Run("WithContext", func(t *testing.T) {
		ctx := context.Background()
		newLogger := logger.(*ZapLogger).WithContext(ctx)

		require.NotNil(t, newLogger)
		assert.Equal(t, Info, newLogger.LogLevel)
	})

	t.Run("WithField", func(t *testing.T) {
		newLogger := logger.(*ZapLogger).WithField("key", "value")

		require.NotNil(t, newLogger)
		assert.Equal(t, Info, newLogger.LogLevel)
	})

	t.Run("WithFields", func(t *testing.T) {
		fields := []zap.Field{
			zap.String("key1", "value1"),
			zap.String("key2", "value2"),
		}
		newLogger := logger.(*ZapLogger).WithFields(fields...)

		require.NotNil(t, newLogger)
		assert.Equal(t, Info, newLogger.LogLevel)
	})

	t.Run("Sugar", func(t *testing.T) {
		sugar := logger.(*ZapLogger).Sugar()

		require.NotNil(t, sugar)
	})
}

func TestZapLogger_SilentLevel(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&buf),
		zapcore.InfoLevel,
	)

	zapLogger := zap.New(core)
	logger := NewZapLogger(zapLogger, Config{
		LogLevel: Silent,
	})

	// Test that nothing is logged at silent level
	logger.Info(ctx, "This should not be logged")
	logger.Warn(ctx, "This should not be logged")
	logger.Error(ctx, "This should not be logged")

	assert.Empty(t, buf.String())
}

func TestZapLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    LogLevel
		expected zapcore.Level
	}{
		{"Silent", Silent, zapcore.DPanicLevel},
		{"Error", Error, zapcore.ErrorLevel},
		{"Warn", Warn, zapcore.WarnLevel},
		{"Info", Info, zapcore.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ZapLevel(tt.level))
		})
	}
}

func TestNewZapLoggerWithConfig(t *testing.T) {
	// Custom config
	config := Config{
		LogLevel:      Info,
		SlowThreshold: 100 * time.Millisecond,
		Colorful:      true,
	}

	customZapConfig := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapcore.InfoLevel),
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger := NewZapLoggerWithConfig(config, customZapConfig)

	require.NotNil(t, logger)
	assert.Equal(t, Info, logger.(*ZapLogger).LogLevel)
	assert.Equal(t, 100*time.Millisecond, logger.(*ZapLogger).SlowThreshold)
}

func BenchmarkZapLogger(b *testing.B) {
	ctx := context.Background()
	var buf bytes.Buffer

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(&buf),
		zapcore.InfoLevel,
	)

	zapLogger := zap.New(core)

	logger := NewZapLogger(zapLogger, Config{
		LogLevel: Info,
	})

	b.Run("Info", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.Info(ctx, "benchmark message", "iteration", i)
		}
	})

	b.Run("Trace", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.Trace(ctx, time.Now(), func() (string, int64) {
				return "SELECT 1", 1
			}, nil)
		}
	})
}

func BenchmarkZapVsNoop(b *testing.B) {
	ctx := context.Background()

	noopLogger := NewZapLogger(zap.NewNop(), Config{LogLevel: Info})

	b.Run("Noop Logger", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			noopLogger.Info(ctx, "benchmark message", "iteration", i)
		}
	})
}
