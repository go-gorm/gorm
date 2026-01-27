package logger

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogrusLogger(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer

	logrusLogger := logrus.New()
	logrusLogger.SetOutput(&buf)

	config := Config{
		LogLevel:      Info,
		SlowThreshold: 100 * time.Millisecond,
		Colorful:      true,
	}

	logrusAdapter := NewLogrusLogger(logrusLogger, config)

	require.NotNil(t, logrusAdapter)
	assert.Equal(t, Info, logrusAdapter.(*LogrusLogger).LogLevel)
	assert.Equal(t, 100*time.Millisecond, logrusAdapter.(*LogrusLogger).SlowThreshold)
}

func TestLogrusLogger_LogMode(t *testing.T) {
	logrusLogger := logrus.New()
	logrusLogger.SetOutput(&bytes.Buffer{})

	logger := NewLogrusLogger(logrusLogger, Config{
		LogLevel: Error,
	})

	// Test changing log mode
	infoLogger := logger.LogMode(Info)
	assert.Equal(t, Info, infoLogger.(*LogrusLogger).LogLevel)

	// Test that original is not affected
	assert.Equal(t, Error, logger.(*LogrusLogger).LogLevel)
}

func TestLogrusLogger_LogLevels(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer

	logrusLogger := logrus.New()
	logrusLogger.SetOutput(&buf)
	logger := NewLogrusLogger(logrusLogger, Config{
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

func TestLogrusLogger_Trace(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer

	logrusLogger := logrus.New()
	logrusLogger.SetOutput(&buf)
	logger := NewLogrusLogger(logrusLogger, Config{
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
		logger.(*LogrusLogger).IgnoreRecordNotFoundError = true

		logger.Trace(ctx, time.Now(), func() (string, int64) {
			return "SELECT * FROM empty_table", 0
		}, ErrRecordNotFound)

		output := buf.String()
		// Should not log when record not found error is ignored
		if logger.(*LogrusLogger).IgnoreRecordNotFoundError {
			assert.Empty(t, output)
		}
	})
}

func TestLogrusLogger_ParamsFilter(t *testing.T) {
	ctx := context.Background()
	logrusLogger := logrus.New()
	logrusLogger.SetOutput(&bytes.Buffer{})

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
			logger := NewLogrusLogger(logrusLogger, Config{
				ParameterizedQueries: tt.parameterized,
			})

			sql, params := logger.(*LogrusLogger).ParamsFilter(ctx, tt.sql, tt.params...)

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

func TestLogrusLogger_WithMethods(t *testing.T) {
	logrusLogger := logrus.New()
	logrusLogger.SetOutput(&bytes.Buffer{})

	logger := NewLogrusLogger(logrusLogger, Config{
		LogLevel: Info,
	})

	t.Run("WithContext", func(t *testing.T) {
		ctx := context.Background()
		newLogger := logger.(*LogrusLogger).WithContext(ctx)

		require.NotNil(t, newLogger)
		assert.Equal(t, Info, newLogger.LogLevel)
	})

	t.Run("WithField", func(t *testing.T) {
		newLogger := logger.(*LogrusLogger).WithField("key", "value")

		require.NotNil(t, newLogger)
		assert.Equal(t, Info, newLogger.LogLevel)
	})

	t.Run("WithFields", func(t *testing.T) {
		fields := logrus.Fields{
			"key1": "value1",
			"key2": "value2",
		}
		newLogger := logger.(*LogrusLogger).WithFields(fields)

		require.NotNil(t, newLogger)
		assert.Equal(t, Info, newLogger.LogLevel)
	})
}

func TestLogrusLogger_SilentLevel(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer

	logrusLogger := logrus.New()
	logrusLogger.SetOutput(&buf)
	logger := NewLogrusLogger(logrusLogger, Config{
		LogLevel: Silent,
	})

	// Test that nothing is logged at silent level
	logger.Info(ctx, "This should not be logged")
	logger.Warn(ctx, "This should not be logged")
	logger.Error(ctx, "This should not be logged")

	assert.Empty(t, buf.String())
}

func TestLogrusLogger_TimestampFormatting(t *testing.T) {
	var buf bytes.Buffer

	logrusLogger := logrus.New()
	logrusLogger.SetOutput(&buf)
	logrusLogger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	logger := NewLogrusLogger(logrusLogger, Config{
		LogLevel: Info,
	})

	ctx := context.Background()
	logger.Info(ctx, "Test message with timestamp")

	output := buf.String()
	assert.Contains(t, output, "Test message with timestamp")
}

func BenchmarkLogrusLogger(b *testing.B) {
	ctx := context.Background()
	var buf bytes.Buffer

	logrusLogger := logrus.New()
	logrusLogger.SetOutput(&buf)

	logger := NewLogrusLogger(logrusLogger, Config{
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
