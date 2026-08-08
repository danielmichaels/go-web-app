package logging

import (
	"bytes"
	"context"
	"log/slog"
	"testing"
	"testing/slogtest"
)

func group(t *testing.T, entry map[string]any, name string) map[string]any {
	t.Helper()
	inner, ok := entry[name].(map[string]any)
	if !ok {
		t.Fatalf("group %q missing or not an object in %v", name, entry)
	}
	return inner
}

// trace_id is what ties a log line back to the request that caused it, so a
// query for one must find every record. Adding it to the record would nest it
// inside whatever group is open, putting some records out of reach.
func TestSlogHandlerTraceIDStaysTopLevelUnderGroup(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(newTestHandler(&buf)).WithGroup("req")

	logger.InfoContext(WithTraceID(context.Background(), "t-1"), "msg", slog.String("path", "/x"))
	entry := decode(t, &buf)

	if entry["trace_id"] != "t-1" {
		t.Errorf("trace_id = %v, want t-1 at the top level: %v", entry["trace_id"], entry)
	}
	if got := group(t, entry, "req")["path"]; got != "/x" {
		t.Errorf("req.path = %v, want /x — record attrs must still be grouped", got)
	}
	if _, nested := group(t, entry, "req")["trace_id"]; nested {
		t.Errorf("trace_id also nested inside the group: %v", entry)
	}
}

func TestSlogHandlerGroupingCombinations(t *testing.T) {
	tests := []struct {
		name   string
		build  func(*slog.Logger) *slog.Logger
		assert func(*testing.T, map[string]any)
	}{
		{
			name:  "attrs before a group stay top level",
			build: func(l *slog.Logger) *slog.Logger { return l.With(slog.String("svc", "api")).WithGroup("g") },
			assert: func(t *testing.T, e map[string]any) {
				if e["svc"] != "api" {
					t.Errorf("svc = %v, want api at top level", e["svc"])
				}
				if got := group(t, e, "g")["path"]; got != "/x" {
					t.Errorf("g.path = %v, want /x", got)
				}
			},
		},
		{
			name:  "attrs after a group are grouped",
			build: func(l *slog.Logger) *slog.Logger { return l.WithGroup("g").With(slog.String("svc", "api")) },
			assert: func(t *testing.T, e map[string]any) {
				if got := group(t, e, "g")["svc"]; got != "api" {
					t.Errorf("g.svc = %v, want api", got)
				}
			},
		},
		{
			name:  "nested groups",
			build: func(l *slog.Logger) *slog.Logger { return l.WithGroup("a").WithGroup("b") },
			assert: func(t *testing.T, e map[string]any) {
				if got := group(t, group(t, e, "a"), "b")["path"]; got != "/x" {
					t.Errorf("a.b.path = %v, want /x", got)
				}
			},
		},
		{
			name:  "empty group name is a no-op",
			build: func(l *slog.Logger) *slog.Logger { return l.WithGroup("") },
			assert: func(t *testing.T, e map[string]any) {
				if e["path"] != "/x" {
					t.Errorf("path = %v, want /x at top level", e["path"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := tt.build(slog.New(newTestHandler(&buf)))

			logger.InfoContext(WithTraceID(context.Background(), "t-1"), "msg", slog.String("path", "/x"))
			entry := decode(t, &buf)

			if entry["trace_id"] != "t-1" {
				t.Errorf("trace_id = %v, want t-1 at the top level: %v", entry["trace_id"], entry)
			}
			tt.assert(t, entry)
		})
	}
}

// Branching a logger must not let one child's attrs leak into its sibling.
func TestSlogHandlerGroupedBranchesAreIndependent(t *testing.T) {
	var buf bytes.Buffer
	base := slog.New(newTestHandler(&buf)).WithGroup("g")

	base.With(slog.String("branch", "left")).InfoContext(context.Background(), "msg")
	left := decode(t, &buf)

	buf.Reset()
	base.With(slog.String("branch", "right")).InfoContext(context.Background(), "msg")
	right := decode(t, &buf)

	if got := group(t, left, "g")["branch"]; got != "left" {
		t.Errorf("left branch = %v, want left", got)
	}
	if got := group(t, right, "g")["branch"]; got != "right" {
		t.Errorf("right branch = %v, want right", got)
	}
}

// The standard library's own conformance suite for slog.Handler
// implementations. It is what catches the contract details a hand-written
// handler is easy to get wrong: empty groups, inline groups, resolving
// LogValuers.
func TestSlogHandlerConformsToSlogContract(t *testing.T) {
	var buf bytes.Buffer
	slogtest.Run(t,
		func(*testing.T) slog.Handler {
			buf.Reset()
			return &SlogHandler{inner: slog.NewJSONHandler(&buf, nil)}
		},
		func(t *testing.T) map[string]any { return decode(t, &buf) },
	)
}
