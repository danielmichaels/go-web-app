package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/danielmichaels/go-web-app/internal/tracing"
)

func TestSlogHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	jsonHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	handler := &SlogHandler{Handler: jsonHandler}

	attrs := []slog.Attr{
		slog.String("service", "test-service"),
	}

	newHandler := handler.WithAttrs(attrs)

	if _, ok := newHandler.(*SlogHandler); !ok {
		t.Error("WithAttrs should return a *SlogHandler")
	}
}

func TestSlogHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	jsonHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	handler := &SlogHandler{Handler: jsonHandler}

	newHandler := handler.WithGroup("test-group")

	if _, ok := newHandler.(*SlogHandler); !ok {
		t.Error("WithGroup should return a *SlogHandler")
	}
}

func TestSlogHandler_Handle(t *testing.T) {
	var buf bytes.Buffer
	jsonHandler := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	handler := &SlogHandler{Handler: jsonHandler}
	logger := slog.New(handler)

	traceID := "test-trace-123"
	ctx := context.WithValue(context.Background(), tracing.TraceCtxKey, traceID)

	logger.InfoContext(ctx, "test message", "key", "value")

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if gotTraceID, ok := logEntry["trace_id"]; !ok || gotTraceID != traceID {
		t.Errorf("Expected trace_id=%s, got %v", traceID, gotTraceID)
	}
}
