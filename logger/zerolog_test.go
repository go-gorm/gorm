package logger

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestZerolog() (zerolog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	return zerolog.New(&buf).With().Timestamp().Logger(), &buf
}

func TestNewZerologLogger(t *testing.T) {
	zerologLogger, buf := setupTestZerolog()

	config := Config{
		LogLevel:      Info,
		SlowThreshold: 100 * time.Millisecond,
		Colorful:      true,
	}

	zerologAdapter := NewZerologLogger(zerologLogger, config)

	require.NotNil(t, zerologAdapter)
	assert.Equal(t, Info, zerologAdapter.(*ZerologLogger).LogLevel)
	assert.Equal(t, 100*time.Millisecond, zerologAdapter.(*ZerologLogger).SlowThreshold)
	require.NotNil(t, buf)
}

func TestZerologLogger_LogMode(t *testing.T) {
	zerologLogger, _ := setupTestZerolog()

	logger := NewZerologLogger(zerologLogger, Config{
		LogLevel: Error,
	})

	// Test changing log mode
	infoLogger := logger.LogMode(Info)
	assert.Equal(t, Info, infoLogger.(*ZerologLogger).LogLevel)

	// Test that original is not affected
	assert.Equal(t, Error, logger.(*ZerologLogger).LogLevel)
}

func TestZerologLogger_LogLevels(t *testing.T) {
	ctx := context.Background()

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
			zerologLogger, testBuf := setupTestZerolog()
			testLogger := NewZerologLogger(zerologLogger, Config{
				LogLevel: tt.level,
			})

			switch tt.level {
			case Info:
				testLogger.Info(ctx, tt.logMsg, "key", "value")
			case Warn:
				testLogger.Warn(ctx, tt.logMsg, "key", "value")
			case Error:
				testLogger.Error(ctx, tt.logMsg, "key", "value")
			}

			output := testBuf.String()
			assert.Contains(t, output, tt.logMsg)
			assert.Contains(t, output, "key")
			assert.Contains(t, output, "value")
		})
	}
}

func TestZerologLogger_Trace(t *testing.T) {
	ctx := context.Background()

	t.Run("Normal trace", func(t *testing.T) {
		zerologLogger, testBuf := setupTestZerolog()
		testLogger := NewZerologLogger(zerologLogger, Config{
			LogLevel: Info,
		})
		testLogger.Trace(ctx, time.Now(), func() (string, int64) {
			return "SELECT * FROM users WHERE id = ?", 5
		}, nil)

		output := testBuf.String()
		assert.Contains(t, output, "SELECT * FROM users WHERE id = ?")
		assert.Contains(t, output, "rows")
		assert.Contains(t, output, "5")
		assert.Contains(t, output, "duration")
	})

	t.Run("Slow query", func(t *testing.T) {
		zerologLogger, testBuf := setupTestZerolog()
		testLogger := NewZerologLogger(zerologLogger, Config{
			LogLevel:      Info,
			SlowThreshold: 100 * time.Millisecond,
		})
		testLogger.Trace(ctx, time.Now().Add(-150*time.Millisecond), func() (string, int64) {
			return "SELECT * FROM large_table", 1000
		}, nil)

		output := testBuf.String()
		assert.Contains(t, output, "SELECT * FROM large_table")
		assert.Contains(t, output, "slow_threshold")
	})

	t.Run("Error trace", func(t *testing.T) {
		zerologLogger, testBuf := setupTestZerolog()
		testLogger := NewZerologLogger(zerologLogger, Config{
			LogLevel: Error,
		})
		testError := assert.AnError
		testLogger.Trace(ctx, time.Now(), func() (string, int64) {
			return "SELECT * FROM non_existent_table", 0
		}, testError)

		output := testBuf.String()
		assert.Contains(t, output, "SELECT * FROM non_existent_table")
		assert.Contains(t, output, "error")
	})

	t.Run("Record not found error with ignore", func(t *testing.T) {
		zerologLogger, testBuf := setupTestZerolog()
		testLogger := NewZerologLogger(zerologLogger, Config{
			LogLevel:                  Error,
			IgnoreRecordNotFoundError: true,
		})
		testLogger.Trace(ctx, time.Now(), func() (string, int64) {
			return "SELECT * FROM empty_table", 0
		}, ErrRecordNotFound)

		output := testBuf.String()
		// Should not log when record not found error is ignored
		if testLogger.(*ZerologLogger).IgnoreRecordNotFoundError {
			assert.Empty(t, output)
		}
	})
}

func TestZerologLogger_ParamsFilter(t *testing.T) {
	ctx := context.Background()
	zerologLogger, _ := setupTestZerolog()

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
			logger := NewZerologLogger(zerologLogger, Config{
				ParameterizedQueries: tt.parameterized,
			})

			sql, params := logger.(*ZerologLogger).ParamsFilter(ctx, tt.sql, tt.params...)

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

func TestZerologLogger_WithMethods(t *testing.T) {
	zerologLogger, _ := setupTestZerolog()

	logger := NewZerologLogger(zerologLogger, Config{
		LogLevel: Info,
	})

	t.Run("WithContext", func(t *testing.T) {
		ctx := context.Background()
		newLogger := logger.(*ZerologLogger).WithContext(ctx)

		require.NotNil(t, newLogger)
		assert.Equal(t, Info, newLogger.LogLevel)
	})

	t.Run("With", func(t *testing.T) {
		context := logger.(*ZerologLogger).With()

		require.NotNil(t, context)
	})

	t.Run("Level", func(t *testing.T) {
		newLogger := logger.(*ZerologLogger).Level(zerolog.InfoLevel)

		require.NotNil(t, newLogger)
		assert.Equal(t, Info, logger.(*ZerologLogger).LogLevel)
	})

	t.Run("Sample", func(t *testing.T) {
		sampler := &zerolog.BasicSampler{N: 10}
		newLogger := logger.(*ZerologLogger).Sample(sampler)

		require.NotNil(t, newLogger)
		assert.Equal(t, Info, logger.(*ZerologLogger).LogLevel)
	})

	t.Run("Hook", func(t *testing.T) {
		hook := zerolog.LevelHook{}
		newLogger := logger.(*ZerologLogger).Hook(hook)

		require.NotNil(t, newLogger)
		assert.Equal(t, Info, logger.(*ZerologLogger).LogLevel)
	})
}

func TestZerologLogger_SilentLevel(t *testing.T) {
	ctx := context.Background()
	zerologLogger, buf := setupTestZerolog()
	logger := NewZerologLogger(zerologLogger, Config{
		LogLevel: Silent,
	})

	// Test that nothing is logged at silent level
	logger.Info(ctx, "This should not be logged")
	logger.Warn(ctx, "This should not be logged")
	logger.Error(ctx, "This should not be logged")

	assert.Empty(t, buf.String())
}

func TestZerologLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    LogLevel
		expected zerolog.Level
	}{
		{"Silent", Silent, zerolog.NoLevel},
		{"Error", Error, zerolog.ErrorLevel},
		{"Warn", Warn, zerolog.WarnLevel},
		{"Info", Info, zerolog.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ZerologLevel(tt.level))
		})
	}
}

func TestNewZerologLoggerWithConfig(t *testing.T) {
	var buf bytes.Buffer

	// Custom config
	config := Config{
		LogLevel:      Info,
		SlowThreshold: 100 * time.Millisecond,
		Colorful:      true,
	}

	ctx := zerolog.New(&buf).With().Timestamp()
	logger := NewZerologLoggerWithConfig(config, ctx)

	require.NotNil(t, logger)
	assert.Equal(t, Info, logger.(*ZerologLogger).LogLevel)
	assert.Equal(t, 100*time.Millisecond, logger.(*ZerologLogger).SlowThreshold)
}

func BenchmarkZerologLogger(b *testing.B) {
	ctx := context.Background()
	zerologLogger, _ := setupTestZerolog()

	logger := NewZerologLogger(zerologLogger, Config{
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

func BenchmarkZerologVsNoop(b *testing.B) {
	ctx := context.Background()
	zerologLogger := zerolog.Nop()
	noopLogger := NewZerologLogger(zerologLogger, Config{LogLevel: Info})

	b.Run("Noop Logger", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			noopLogger.Info(ctx, "benchmark message", "iteration", i)
		}
	})
}
