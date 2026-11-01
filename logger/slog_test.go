//go:build go1.21

package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestSlogLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	handler := slog.NewTextHandler(buf, &slog.HandlerOptions{AddSource: true})
	logger := NewSlogLogger(slog.New(handler), Config{LogLevel: Info})

	logger.Trace(context.Background(), time.Now(), func() (string, int64) {
		return "select count(*) from users", 0
	}, nil)

	if strings.Contains(buf.String(), "gorm/logger/slog.go") {
		t.Error("Found internal slog.go reference in caller frame. Expected only test file references.")
	}

	if !strings.Contains(buf.String(), "gorm/logger/slog_test.go") {
		t.Error("Missing expected test file reference. 'gorm/logger/slog_test.go' should appear in caller frames.")
	}
}

// unencodableError can't be encoded as JSON (like pgx's ConnectError, which
// holds a DialFunc), so it must reach the handler only as formatted text.
type unencodableError struct {
	dial func()
}

func (unencodableError) Error() string { return "dial tcp 127.0.0.1:5432: connection refused" }

func TestSlogLogger_FormatsMessages(t *testing.T) {
	tests := []struct {
		level string
		log   func(Interface, string, ...interface{})
	}{
		{"INFO", func(l Interface, msg string, data ...interface{}) { l.Info(context.Background(), msg, data...) }},
		{"WARN", func(l Interface, msg string, data ...interface{}) { l.Warn(context.Background(), msg, data...) }},
		{"ERROR", func(l Interface, msg string, data ...interface{}) { l.Error(context.Background(), msg, data...) }},
	}
	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := NewSlogLogger(slog.New(slog.NewJSONHandler(buf, nil)), Config{LogLevel: Info})

			tt.log(logger, "failed to initialize database, got error %v", unencodableError{dial: func() {}})

			var entry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
				t.Fatalf("expected one JSON entry, got %q", buf.String())
			}
			expected := "failed to initialize database, got error dial tcp 127.0.0.1:5432: connection refused"
			if entry["msg"] != expected {
				t.Errorf("expected msg %q, got %q", expected, entry["msg"])
			}
			if entry["level"] != tt.level {
				t.Errorf("expected level %q, got %q", tt.level, entry["level"])
			}
		})
	}
}
