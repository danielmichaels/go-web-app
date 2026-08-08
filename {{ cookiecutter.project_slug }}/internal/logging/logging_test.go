package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	"{{ cookiecutter.go_module_path.strip() }}/internal/config"

	"github.com/go-chi/chi/v5/middleware"
)

func newTestHandler(buf *bytes.Buffer) *SlogHandler {
	return &SlogHandler{
		inner: slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}),
	}
}

func decode(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("parse log output %q: %v", buf.String(), err)
	}
	return entry
}

func TestSlogHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	handler := newTestHandler(&buf)

	newHandler := handler.WithAttrs([]slog.Attr{slog.String("service", "test-service")})

	if _, ok := newHandler.(*SlogHandler); !ok {
		t.Fatal("WithAttrs should return a *SlogHandler, or the trace_id wrapper is lost")
	}
	slog.New(newHandler).InfoContext(WithTraceID(context.Background(), "t-1"), "msg")
	entry := decode(t, &buf)
	if entry["service"] != "test-service" {
		t.Errorf("service = %v, want test-service", entry["service"])
	}
	if entry["trace_id"] != "t-1" {
		t.Errorf("trace_id = %v, want t-1", entry["trace_id"])
	}
}

func TestSlogHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	handler := newTestHandler(&buf)

	newHandler := handler.WithGroup("test-group")

	if _, ok := newHandler.(*SlogHandler); !ok {
		t.Fatal("WithGroup should return a *SlogHandler, or the trace_id wrapper is lost")
	}
}

func TestSlogHandler_HandleAddsTraceID(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(newTestHandler(&buf))

	logger.InfoContext(WithTraceID(context.Background(), "test-trace-123"), "test message")

	if got := decode(t, &buf)["trace_id"]; got != "test-trace-123" {
		t.Errorf("trace_id = %v, want test-trace-123", got)
	}
}

// A trace id set explicitly must win over chi's request id, so a value carried
// across a service boundary is not silently replaced by a per-request one.
func TestSlogHandler_HandleFallsBackToRequestID(t *testing.T) {
	tests := []struct {
		name string
		ctx  func() context.Context
		want any
	}{
		{
			name: "request id only",
			ctx: func() context.Context {
				return context.WithValue(context.Background(), middleware.RequestIDKey, "req-9")
			},
			want: "req-9",
		},
		{
			name: "explicit trace id wins",
			ctx: func() context.Context {
				ctx := context.WithValue(context.Background(), middleware.RequestIDKey, "req-9")
				return WithTraceID(ctx, "trace-1")
			},
			want: "trace-1",
		},
		{
			name: "neither present",
			ctx:  context.Background,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			slog.New(newTestHandler(&buf)).InfoContext(tt.ctx(), "msg")
			if got := decode(t, &buf)["trace_id"]; got != tt.want {
				t.Errorf("trace_id = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetupLoggerJSONHonoursLevel(t *testing.T) {
	var buf bytes.Buffer
	cfg := &config.Conf{}
	cfg.AppConf.LogJson = true
	cfg.AppConf.LogLevel = slog.LevelWarn

	logger := SetupLogger(cfg, WithOutput(&buf))

	logger.Info("dropped")
	if buf.Len() != 0 {
		t.Fatalf("info record logged at warn level: %q", buf.String())
	}

	logger.WarnContext(WithTraceID(context.Background(), "t-2"), "kept")
	entry := decode(t, &buf)
	if entry["body"] != "kept" {
		t.Errorf("body = %v, want kept — SetupLogger must install the schema", entry["body"])
	}
	if entry["trace_id"] != "t-2" {
		t.Errorf("trace_id = %v, want t-2 — SetupLogger must wrap with SlogHandler", entry["trace_id"])
	}
}
