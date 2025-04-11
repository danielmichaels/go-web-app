package tracing

import (
	"context"

	"github.com/rs/xid"
)

type contextKey int

const (
	TraceCtxKey contextKey = iota + 1
)

// WithNewTraceID creates a new context with a unique trace ID if one does not already exist.
// If a trace ID already exists in the provided context, the original context is returned.
func WithNewTraceID(ctx context.Context, regenerate bool) context.Context {
	// Check if trace ID already exists
	if traceID, ok := ctx.Value(TraceCtxKey).(string); ok && traceID != "" && !regenerate {
		return ctx
	}
	// Create new trace ID if none exists
	guid := xid.New()
	return context.WithValue(ctx, TraceCtxKey, guid.String())
}
