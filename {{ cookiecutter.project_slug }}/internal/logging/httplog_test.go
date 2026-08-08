package logging

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"{{ cookiecutter.go_module_path.strip() }}/internal/config"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httplog/v3"
)

func jsonLogger(t *testing.T, buf *bytes.Buffer) *slog.Logger {
	t.Helper()
	cfg := &config.Conf{}
	cfg.AppConf.LogJson = true
	cfg.AppConf.LogLevel = slog.LevelDebug
	return SetupLogger(cfg, WithOutput(buf))
}

// The schema has to be installed on the handler as well as on the middleware.
// With only the middleware half wired up the attributes are OTEL-named but
// slog's own keys are not, which is not a format any collector understands.
func TestSetupLoggerJSONUsesSchemaFieldNames(t *testing.T) {
	var buf bytes.Buffer
	jsonLogger(t, &buf).Info("hello")
	entry := decode(t, &buf)

	for _, key := range []string{"timestamp", "severity_text", "body"} {
		if _, ok := entry[key]; !ok {
			t.Errorf("missing OTEL key %q in %v", key, entry)
		}
	}
	for _, key := range []string{"time", "level", "msg"} {
		if _, ok := entry[key]; ok {
			t.Errorf("slog key %q survived, want it renamed: %v", key, entry)
		}
	}
}

// httplog reports a client disconnect as slog.Any("error", ...), which the
// schema is what rewrites to error.message.
func TestSetupLoggerJSONRenamesErrorKey(t *testing.T) {
	var buf bytes.Buffer
	jsonLogger(t, &buf).Error("boom", slog.Any(httplog.ErrorKey, errors.New("aborted")))
	entry := decode(t, &buf)

	if _, ok := entry["error.message"]; !ok {
		t.Errorf("error not renamed to error.message: %v", entry)
	}
}

// httplog's own ReplaceAttr formats at RFC3339, which rounds to the second and
// makes same-second records unorderable.
func TestOTELReplaceAttrKeepsSubSecondPrecision(t *testing.T) {
	ts := time.Date(2026, 8, 8, 10, 30, 0, 123456789, time.UTC)

	got := otelReplaceAttr(nil, slog.Time(slog.TimeKey, ts))

	if got.Key != "timestamp" {
		t.Errorf("key = %q, want timestamp", got.Key)
	}
	if want := ts.Format(time.RFC3339Nano); got.Value.String() != want {
		t.Errorf("value = %q, want %q", got.Value.String(), want)
	}
}

func routeCtx(pattern string) context.Context {
	rctx := chi.NewRouteContext()
	if pattern != "" {
		rctx.RoutePatterns = []string{pattern}
	}
	return context.WithValue(context.Background(), chi.RouteCtxKey, rctx)
}

// A 404 from an unmatched route is bot noise; a 404 from a route that matched
// is a genuinely missing resource. Only the first is demoted, and a request
// that never went through the mux at all must not be touched.
func TestSlogHandlerDemotesUnmatchedRouteWarnings(t *testing.T) {
	tests := []struct {
		name  string
		ctx   context.Context
		level slog.Level
		want  string
	}{
		{"unmatched route", routeCtx(""), slog.LevelWarn, "INFO"},
		{"matched route", routeCtx("/widgets/{id}"), slog.LevelWarn, "WARN"},
		{"no route context", context.Background(), slog.LevelWarn, "WARN"},
		{"errors are never demoted", routeCtx(""), slog.LevelError, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(newTestHandler(&buf))

			logger.Log(tt.ctx, tt.level, "msg")

			if got := decode(t, &buf)["level"]; got != tt.want {
				t.Errorf("level = %v, want %v", got, tt.want)
			}
		})
	}
}

// The whole seam end to end: chi decides whether a route matched, httplog
// decides the level from the status, and the handler demotes the noise.
func TestAccessLogLevelThroughChi(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"unmatched route is routine", "/wp-admin", "INFO"},
		{"matched route returning 404 still warns", "/widgets/9", "WARN"},
		{"success is routine", "/widgets", "INFO"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := jsonLogger(t, &buf)

			r := chi.NewMux()
			r.Use(httplog.RequestLogger(logger, &httplog.Options{
				Level:  slog.LevelInfo,
				Schema: AccessLogSchema,
			}))
			r.Get("/widgets", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
			r.Get("/widgets/{id}", func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "no such widget", http.StatusNotFound)
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			r.ServeHTTP(httptest.NewRecorder(), req)

			if got := decode(t, &buf)["severity_text"]; got != tt.want {
				t.Errorf("severity_text = %v, want %v (%q)", got, tt.want, buf.String())
			}
		})
	}
}
