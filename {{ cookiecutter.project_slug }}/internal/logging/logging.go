// Package logging builds the application slog.Logger from config and owns the
// field-name schema shared with the access log.
//
// Access logging itself is go-chi/httplog's, wired up in internal/server. The
// schema has to be installed on both halves — the handler here, the middleware
// there — or records come out with OTEL attributes under slog's own key names.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"time"

	"{{ cookiecutter.go_module_path.strip() }}/internal/config"

	"github.com/SladkyCitron/slogcolor"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v3"
)

// AccessLogSchema is the field-name schema for both the application logger and
// the access log. internal/server passes it to httplog; keep it the one source
// of truth so the two halves cannot drift apart.
var AccessLogSchema = httplog.SchemaOTEL

// otelReplaceAttr renames slog's built-in keys to AccessLogSchema. The
// timestamp is handled here rather than by the schema because httplog formats
// it at RFC3339's one-second resolution, which leaves same-second records
// unorderable.
func otelReplaceAttr(groups []string, a slog.Attr) slog.Attr {
	if len(groups) == 0 && a.Key == slog.TimeKey {
		return slog.String(AccessLogSchema.Timestamp, a.Value.Time().Format(time.RFC3339Nano))
	}
	return AccessLogSchema.ReplaceAttr(groups, a)
}

// Option customises logger construction.
type Option func(*options)

type options struct{ out io.Writer }

// WithOutput redirects log output away from stderr. Tests use it to capture
// records.
func WithOutput(w io.Writer) Option { return func(o *options) { o.out = w } }

type traceIDKey struct{}

// WithTraceID stores id on ctx so every record logged with that context
// carries it, not only the ones written by request middleware.
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey{}, id)
}

func TraceID(ctx context.Context) string {
	if id, ok := ctx.Value(traceIDKey{}).(string); ok {
		return id
	}
	return ""
}

// SlogHandler stamps trace_id onto every record whose context carries one, so
// a log line written deep in a service call can still be tied back to the
// request that caused it.
type SlogHandler struct {
	inner slog.Handler
}

func (h *SlogHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *SlogHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level == slog.LevelWarn && unmatchedRoute(ctx) {
		r.Level = slog.LevelInfo
		if !h.inner.Enabled(ctx, r.Level) {
			return nil
		}
	}
	traceID := TraceID(ctx)
	if traceID == "" {
		traceID = middleware.GetReqID(ctx)
	}
	if traceID != "" {
		// Cloned because AddAttrs can append into a backing array the caller
		// still shares, which would corrupt a sibling handler's copy.
		r = r.Clone()
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

// unmatchedRoute reports whether ctx belongs to a request that reached the mux
// and matched nothing, which httplog would otherwise log at warn. A handler
// answering 404 for a missing resource is a real warning and keeps its level;
// so does anything with no route context at all, such as a background job.
func unmatchedRoute(ctx context.Context) bool {
	rctx := chi.RouteContext(ctx)
	return rctx != nil && rctx.RoutePattern() == ""
}

// SetupLogger returns the application logger: JSON for deployments, coloured
// single lines for local dev.
func SetupLogger(cfg *config.Conf, opts ...Option) *slog.Logger {
	o := options{out: os.Stderr}
	for _, fn := range opts {
		fn(&o)
	}

	var handler slog.Handler
	if cfg.AppConf.LogJson {
		handler = slog.NewJSONHandler(o.out, &slog.HandlerOptions{
			Level:       cfg.AppConf.LogLevel,
			ReplaceAttr: otelReplaceAttr,
		})
	} else {
		// Built from DefaultOptions rather than a bare literal: the zero value
		// blanks TimeFormat and the output loses its timestamps.
		colorOpts := *slogcolor.DefaultOptions
		colorOpts.Level = cfg.AppConf.LogLevel
		handler = slogcolor.NewHandler(o.out, &colorOpts)
	}
	return slog.New(&SlogHandler{inner: handler})
}
