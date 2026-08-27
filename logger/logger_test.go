package logger

import (
	"context"
	"io"
	"log"
	"testing"
	"time"
)

// A recorder created for a logger must apply that logger's own parameter filter,
// so a logger configured with ParameterizedQueries still hides values while its
// statements are being recorded (as happens during DB.Scan).
func TestRecorderUsesRecordedLoggerParamsFilter(t *testing.T) {
	discard := log.New(io.Discard, "", 0)

	for _, tc := range []struct {
		name                 string
		parameterizedQueries bool
		wantParams           bool
	}{
		{name: "parameterized", parameterizedQueries: true, wantParams: false},
		{name: "not parameterized", parameterizedQueries: false, wantParams: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := NewRecorder(New(discard, Config{
				LogLevel:             Info,
				ParameterizedQueries: tc.parameterizedQueries,
			}))

			_, params := recorder.ParamsFilter(context.Background(), "SELECT * FROM users WHERE name = ?", "secret")
			if got := len(params) > 0; got != tc.wantParams {
				t.Errorf("params returned = %v, want %v", got, tc.wantParams)
			}
		})
	}
}

// A logger that does not implement the filter must still fall back to
// RecorderParamsFilter, so existing users of that hook are unaffected.
func TestRecorderFallsBackToRecorderParamsFilter(t *testing.T) {
	original := RecorderParamsFilter
	t.Cleanup(func() { RecorderParamsFilter = original })

	called := false
	RecorderParamsFilter = func(ctx context.Context, sql string, params ...interface{}) (string, []interface{}) {
		called = true
		return sql, nil
	}

	// plainLogger implements Interface without ParamsFilter.
	recorder := NewRecorder(plainLogger{})

	_, params := recorder.ParamsFilter(context.Background(), "SELECT * FROM users WHERE name = ?", "secret")
	if !called {
		t.Error("expected RecorderParamsFilter to be used for a logger without ParamsFilter")
	}
	if len(params) != 0 {
		t.Errorf("expected the fallback to drop params, got %v", params)
	}
}

type plainLogger struct{}

func (plainLogger) LogMode(LogLevel) Interface                    { return plainLogger{} }
func (plainLogger) Info(context.Context, string, ...interface{})  {}
func (plainLogger) Warn(context.Context, string, ...interface{})  {}
func (plainLogger) Error(context.Context, string, ...interface{}) {}
func (plainLogger) Trace(context.Context, time.Time, func() (string, int64), error) {
}
