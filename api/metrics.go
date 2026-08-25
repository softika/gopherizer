package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// routeUnmatched buckets requests that matched no route.
//
// Labelling those with the raw path would mint a new time series for every URL
// a scanner probes, which is a cheap way to exhaust the metrics backend.
const routeUnmatched = "unmatched"

// metrics holds the HTTP collectors and their registry.
//
// A private registry keeps the collectors out of prometheus' global default,
// so tests can build independent instances without duplicate registration.
type metrics struct {
	registry *prometheus.Registry
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

func newMetrics() *metrics {
	labels := []string{"method", "route", "status"}

	m := &metrics{
		registry: prometheus.NewRegistry(),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		}, labels),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		}, labels),
	}

	m.registry.MustRegister(
		m.requests,
		m.duration,
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	return m
}

// middleware records the outcome and latency of every request.
func (m *metrics) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		// The pattern is only known once routing has happened.
		labels := prometheus.Labels{
			"method": r.Method,
			"route":  routePattern(r),
			"status": strconv.Itoa(ww.Status()),
		}

		m.requests.With(labels).Inc()
		m.duration.With(labels).Observe(time.Since(start).Seconds())
	})
}

// handler serves the collected metrics in the Prometheus exposition format.
func (m *metrics) handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// routePattern returns the matched route template, never the raw path, so path
// parameters cannot become label values.
func routePattern(r *http.Request) string {
	rctx := chi.RouteContext(r.Context())
	if rctx == nil {
		return routeUnmatched
	}

	if pattern := rctx.RoutePattern(); pattern != "" {
		return pattern
	}

	return routeUnmatched
}
