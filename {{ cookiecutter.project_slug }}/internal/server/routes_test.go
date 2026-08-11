package server_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"{{ cookiecutter.go_module_path.strip() }}/internal/config"
	"{{ cookiecutter.go_module_path.strip() }}/internal/server"
	"{{ cookiecutter.go_module_path.strip() }}/internal/testhelpers"
)

func TestMain(m *testing.M) {
	os.Exit(testhelpers.RunTestMain(m))
}

// newTestServer exercises the real Routes() stack against a real database, so
// the middleware order under test is the one that ships.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	ctx := context.Background()
	pg := testhelpers.Shared(ctx, t)
	pg.TruncateAll(ctx, t)

	t.Setenv("DATABASE_URL", pg.DSN)
{% if not cookiecutter.api_only -%}
	t.Setenv("TRUSTED_ORIGINS", "https://trusted.example")
{% endif -%}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	app := server.New(server.Deps{
		Conf:    cfg,
		Log:     slog.New(slog.DiscardHandler),
		Db:      pg.Queries,
		PgxPool: pg.Pool,
	})

	ts := httptest.NewServer(app.Routes())
	t.Cleanup(ts.Close)
	return ts
}

func TestMonitoringEndpoints(t *testing.T) {
	ts := newTestServer(t)

	for _, path := range []string{"/healthz", "/version"} {
		t.Run(path, func(t *testing.T) {
			res, err := ts.Client().Get(ts.URL + path)
			if err != nil {
				t.Fatalf("get %s: %v", path, err)
			}
			defer res.Body.Close()
			if res.StatusCode != http.StatusOK {
				t.Errorf("status = %d, want 200", res.StatusCode)
			}
		})
	}
}

// Metrics ride a listener of their own so the port carrying them can stay off
// the public interface. Answering here would put process statistics and the
// whole route table back in front of anyone who asks.
func TestMetricsAreNotOnThePublicRouter(t *testing.T) {
	ts := newTestServer(t)

	res, err := ts.Client().Get(ts.URL + "/metrics")
	if err != nil {
		t.Fatalf("get /metrics: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", res.StatusCode)
	}
}

// Two headers, not one: huma serves its docs page with a Content-Security-Policy
// of its own and setting that header replaces ours, so X-Frame-Options is the
// only framing protection /docs keeps.
func TestFramingIsDenied(t *testing.T) {
	ts := newTestServer(t)

	for _, path := range []string{"/healthz", "/docs"} {
		t.Run(path, func(t *testing.T) {
			res, err := ts.Client().Get(ts.URL + path)
			if err != nil {
				t.Fatalf("get %s: %v", path, err)
			}
			defer res.Body.Close()

			if got := res.Header.Get("X-Frame-Options"); got != "DENY" {
				t.Errorf("X-Frame-Options = %q, want DENY", got)
			}
		})
	}
}

func TestContentSecurityPolicyDeniesFrameAncestors(t *testing.T) {
	ts := newTestServer(t)

	res, err := ts.Client().Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("get /healthz: %v", err)
	}
	defer res.Body.Close()

	got := res.Header.Get("Content-Security-Policy")
	if !strings.Contains(got, "frame-ancestors 'none'") {
		t.Errorf("Content-Security-Policy = %q, want frame-ancestors 'none'", got)
	}
}

// The API reference is huma's own Scalar renderer rather than a hand-written
// page. The bundle URL is what actually distinguishes it from the Stoplight
// default, and the integrity hash is the reason to prefer huma's page over a
// hand-rolled one at all.
func TestDocsAreRenderedByScalar(t *testing.T) {
	ts := newTestServer(t)

	res, err := ts.Client().Get(ts.URL + "/docs")
	if err != nil {
		t.Fatalf("get /docs: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(body), "@scalar/api-reference") {
		t.Error("/docs does not load the Scalar bundle")
	}
	if !strings.Contains(string(body), "integrity=") {
		t.Error("Scalar bundle loaded without a subresource-integrity hash")
	}
	if csp := res.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'none'") {
		t.Errorf("Content-Security-Policy = %q, want a restrictive default-src", csp)
	}
}

{% if not cookiecutter.api_only -%}
func TestHomePageRenders(t *testing.T) {
	ts := newTestServer(t)

	res, err := ts.Client().Get(ts.URL + "/app")
	if err != nil {
		t.Fatalf("get /app: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("content-type = %q, want text/html", ct)
	}
}

// A cross-origin write must be refused, and refused with 404 rather than 403:
// a 403 would tell a prober the route exists.
func TestCrossOriginPostIsRefusedAsNotFound(t *testing.T) {
	ts := newTestServer(t)

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/app/examples", strings.NewReader("text=x"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Sec-Fetch-Site", "cross-site")

	res, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404 for a cross-origin write", res.StatusCode)
	}
}

// The same-origin equivalent must go through, or the check above would pass
// simply because the route was broken.
func TestSameOriginPostSucceeds(t *testing.T) {
	ts := newTestServer(t)

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/app/examples", strings.NewReader("text=hello"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Sec-Fetch-Site", "same-origin")

	res, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", res.StatusCode)
	}
}
{% endif -%}
