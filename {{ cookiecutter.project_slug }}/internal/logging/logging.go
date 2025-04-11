package logging

import (
	"context"
	"{{ cookiecutter.go_module_path.strip() }}/internal/config"
	"{{ cookiecutter.go_module_path.strip() }}/internal/tracing"
	"io"
	"log/slog"
	"os"

	"github.com/MatusOllah/slogcolor"
)

type SlogHandler struct{ slog.Handler }

func (s *SlogHandler) Handle(ctx context.Context, r slog.Record) error {
	if traceID, ok := ctx.Value(tracing.TraceCtxKey).(string); ok {
		r.Add("trace_id", slog.StringValue(traceID))
	}
	return s.Handler.Handle(ctx, r)
}

func (s *SlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *s
	return &clone
}

func (s *SlogHandler) WithGroup(name string) slog.Handler {
	return &SlogHandler{Handler: s.Handler.WithGroup(name)}
}

func createHandler(writer io.Writer, cfg *config.Conf) slog.Handler {
	if cfg.AppConf.LogJson {
		return slog.NewJSONHandler(writer, &slog.HandlerOptions{
			Level: cfg.AppConf.LogLevel,
		})
	}
	// Use slogcolor for colorful console logging
	return slogcolor.NewHandler(writer, &slogcolor.Options{Level: cfg.AppConf.LogLevel})
}

func SetupLogger(service string, cfg *config.Conf) (*slog.Logger, context.Context) {
	var handler slog.Handler
	handler = createHandler(os.Stderr, cfg)

	handler = &SlogHandler{Handler: handler}

	logger := slog.New(handler)
	// default to none, will be overridden later
	traceID := ""
	ctx := context.WithValue(context.Background(), tracing.TraceCtxKey, traceID)
	return logger, ctx
}
