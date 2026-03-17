package logging

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"{{ cookiecutter.go_module_path.strip() }}/internal/config"

	"github.com/SladkyCitron/slogcolor"
	"github.com/go-chi/chi/v5/middleware"
)

type traceIDKey struct{}

func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, id)
}

func TraceID(ctx context.Context) string {
	if id, ok := ctx.Value(traceIDKey{}).(string); ok {
		return id
	}
	return ""
}

type SlogHandler struct {
	inner slog.Handler
}

func (h *SlogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *SlogHandler) Handle(ctx context.Context, r slog.Record) error {
	traceID := TraceID(ctx)
	if traceID == "" {
		traceID = middleware.GetReqID(ctx)
	}
	if traceID != "" {
		r.AddAttrs(slog.String("trace_id", traceID))
	}
	return h.inner.Handle(ctx, r)
}

func (h *SlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &SlogHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *SlogHandler) WithGroup(name string) slog.Handler {
	return &SlogHandler{inner: h.inner.WithGroup(name)}
}

func SetupLogger(cfg *config.Conf) *slog.Logger {
	var handler slog.Handler
	if cfg.AppConf.LogJson {
		handler = slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: cfg.AppConf.LogLevel,
		})
	} else {
		handler = slogcolor.NewHandler(os.Stderr, &slogcolor.Options{Level: cfg.AppConf.LogLevel})
	}
	return slog.New(&SlogHandler{inner: handler})
}

type responseWriter struct {
	http.ResponseWriter
	status      int
	bytes       int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.status = code
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytes += n
	return n, err
}

func (rw *responseWriter) Flush() {
	if !rw.wroteHeader {
		rw.wroteHeader = true
		rw.status = http.StatusOK
	}
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

type QuietRoute struct {
	Pattern string
	Period  time.Duration
}

type HTTPLoggerConfig struct {
	Logger          *slog.Logger
	Concise         bool
	RequestHeaders  bool
	ResponseHeaders bool
	QuietRoutes     []QuietRoute
	SkipPaths       []string
}

type quietRouteTracker struct {
	lastLogged map[string]time.Time
	mu         sync.Mutex
}

func (q *quietRouteTracker) shouldLog(path string, routes []QuietRoute) bool {
	for _, route := range routes {
		if strings.HasPrefix(path, route.Pattern) {
			q.mu.Lock()
			defer q.mu.Unlock()
			last, ok := q.lastLogged[path]
			if !ok || time.Since(last) >= route.Period {
				q.lastLogged[path] = time.Now()
				return true
			}
			return false
		}
	}
	return true
}

var sensitiveHeaders = map[string]bool{
	"authorization":   true,
	"cookie":          true,
	"set-cookie":      true,
	"x-api-key":       true,
	"x-auth-token":    true,
	"x-csrf-token":    true,
	"x-session-token": true,
	"api-key":         true,
	"apikey":          true,
}

func sanitizeHeaders(headers http.Header) map[string]string {
	result := make(map[string]string, len(headers))
	for key, values := range headers {
		if sensitiveHeaders[strings.ToLower(key)] {
			result[key] = "[REDACTED]"
		} else {
			result[key] = strings.Join(values, ", ")
		}
	}
	return result
}

func statusLevel(status int) slog.Level {
	switch {
	case status >= 500:
		return slog.LevelError
	case status >= 400:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}

func RequestLogger(cfg HTTPLoggerConfig) func(http.Handler) http.Handler {
	tracker := &quietRouteTracker{lastLogged: make(map[string]time.Time)}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, path := range cfg.SkipPaths {
				if strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			start := time.Now()
			wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			reqCtx := r.Context()
			if reqID := middleware.GetReqID(reqCtx); reqID != "" {
				reqCtx = WithTraceID(reqCtx, reqID)
				r = r.WithContext(reqCtx)
			}

			next.ServeHTTP(wrapped, r)
			duration := time.Since(start)

			if !tracker.shouldLog(r.URL.Path, cfg.QuietRoutes) {
				return
			}

			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", wrapped.status),
				slog.String("duration", duration.String()),
				slog.Int("bytes", wrapped.bytes),
				slog.String("remote_addr", r.RemoteAddr),
			}
			if cfg.RequestHeaders {
				attrs = append(attrs, slog.Any("request_headers", sanitizeHeaders(r.Header)))
			}
			if cfg.ResponseHeaders {
				attrs = append(attrs, slog.Any("response_headers", sanitizeHeaders(wrapped.Header())))
			}

			msg := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
			if cfg.Concise {
				msg = fmt.Sprintf("%d %s", wrapped.status, r.URL.Path)
			}

			cfg.Logger.LogAttrs(r.Context(), statusLevel(wrapped.status), msg, attrs...)
		})
	}
}
