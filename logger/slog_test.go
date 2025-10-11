//go:build go1.21

package logger

import (
	"bytes"
	"context"
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
