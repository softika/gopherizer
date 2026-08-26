package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/trace"
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

// metricsOption registers an additional collector on the private registry.
type metricsOption func(*metrics)

func newMetrics(opts ...metricsOption) *metrics {
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

	for _, opt := range opts {
		opt(m)
	}

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
		m.observeDuration(r.Context(), labels, time.Since(start).Seconds())
	})
}

// observeDuration records latency, attaching the active trace as an exemplar.
//
// An exemplar is what turns a latency spike in Grafana into one click through
// to the trace that caused it. Only sampled traces are attached: an unsampled
// trace never reaches the backend, so linking to it would offer a dead end.
func (m *metrics) observeDuration(ctx context.Context, labels prometheus.Labels, seconds float64) {
	observer := m.duration.With(labels)

	if sc := trace.SpanContextFromContext(ctx); sc.IsSampled() {
		if e, ok := observer.(prometheus.ExemplarObserver); ok {
			e.ObserveWithExemplar(seconds, prometheus.Labels{"trace_id": sc.TraceID().String()})
			return
		}
	}

	observer.Observe(seconds)
}

// handler serves the collected metrics.
//
// OpenMetrics is enabled because it is the only exposition format that can
// carry exemplars; Prometheus negotiates it via the Accept header and falls
// back to the classic text format for scrapers that do not ask for it.
func (m *metrics) handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
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
