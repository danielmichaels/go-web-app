// Package logging builds the application slog.Logger from config.
//
// Access logging is not here: go-chi/httplog owns that, wired up in
// internal/server.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"

	"{{ cookiecutter.go_module_path.strip() }}/internal/config"

	"github.com/SladkyCitron/slogcolor"
	"github.com/go-chi/chi/v5/middleware"
)

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
			Level: cfg.AppConf.LogLevel,
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
