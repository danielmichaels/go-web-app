// Package metrics owns the Prometheus registry and the application's bounded
// HTTP metrics. Runtime and process collectors are included so a generated
// app is useful to scrape before it has domain-specific work to measure.
package metrics

import (
	"net/http"
	"strconv"

	"github.com/felixge/httpsnoop"
	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics is an application-local Prometheus registry.
type Metrics struct {
	registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inFlight prometheus.Gauge
}

// New creates a registry with Go runtime, process, and HTTP server metrics.
func New() *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "app",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests handled.",
		}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "app",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds.",
		}, []string{"method", "route"}),
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "app",
			Subsystem: "http",
			Name:      "requests_in_flight",
			Help:      "Current number of HTTP requests in flight.",
		}),
	}
	m.registry.MustRegister(
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
		m.requests,
		m.duration,
		m.inFlight,
	)
	return m
}

// Handler exposes this application's metrics in Prometheus text format.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// Middleware records requests using chi's route pattern. Unmatched paths use a
// fixed label, preventing user-controlled URLs from creating metric series.
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.inFlight.Inc()
		defer m.inFlight.Dec()

		captured := httpsnoop.CaptureMetrics(next, w, r)
		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}
		status := captured.Code
		if status == 0 {
			status = http.StatusOK
		}
		m.requests.WithLabelValues(r.Method, route, strconv.Itoa(status)).Inc()
		m.duration.WithLabelValues(r.Method, route).Observe(captured.Duration.Seconds())
	})
}
