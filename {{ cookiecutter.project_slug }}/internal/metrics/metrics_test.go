package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestHandlerExposesRuntimeMetrics(t *testing.T) {
	m := New()

	res := httptest.NewRecorder()
	m.Handler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if !strings.Contains(res.Body.String(), "go_goroutines") {
		t.Errorf("handler does not expose Go runtime metrics:\n%s", res.Body.String())
	}
}

func TestMiddlewareUsesRoutePatterns(t *testing.T) {
	m := New()
	h := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	r := httptest.NewRequest(http.MethodGet, "/items/123?secret=ignored", nil)
	ctx := chi.NewRouteContext()
	ctx.RoutePatterns = []string{"/items/{id}"}
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, ctx))
	res := httptest.NewRecorder()
	h.ServeHTTP(res, r)

	body := httptest.NewRecorder()
	m.Handler().ServeHTTP(body, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	text := body.Body.String()
	if !strings.Contains(text, `app_http_requests_total{method="GET",route="/items/{id}",status="201"} 1`) {
		t.Fatalf("metrics missing bounded route label:\n%s", text)
	}
	if strings.Contains(text, `route="/items/123"`) || strings.Contains(text, "secret=ignored") {
		t.Fatalf("metrics leaked request URL values:\n%s", text)
	}
}
