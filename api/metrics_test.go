package api

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// routerWithMetrics mounts a parameterised route behind the metrics middleware.
func routerWithMetrics(t *testing.T) (*chi.Mux, *metrics) {
	t.Helper()

	m := newMetrics()
	r := chi.NewRouter()
	r.Use(m.middleware)
	r.Get("/api/v1/profile/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	r.Get("/boom", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	r.Handle("/metrics", m.handler())

	return r, m
}

func scrape(t *testing.T, r http.Handler) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	return w.Body.String()
}

func TestMetricsRecordsRequests(t *testing.T) {
	t.Parallel()

	r, _ := routerWithMetrics(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile/abc", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)

	body := scrape(t, r)

	assert.Contains(t, body, "http_requests_total")
	assert.Contains(t, body, "http_request_duration_seconds")
	assert.Contains(t, body, `method="GET"`)
	assert.Contains(t, body, `status="200"`)
}

// A metric label must never carry a path parameter: one series per profile id
// would blow up cardinality and take the metrics backend down with it.
func TestMetricsUsesRoutePatternNotRawPath(t *testing.T) {
	t.Parallel()

	r, _ := routerWithMetrics(t)

	for _, id := range []string{"aaaa", "bbbb", "cccc"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/profile/"+id, nil)
		r.ServeHTTP(httptest.NewRecorder(), req)
	}

	body := scrape(t, r)

	assert.Contains(t, body, `/api/v1/profile/{id}`, "the route pattern must be the label")

	for _, id := range []string{"aaaa", "bbbb", "cccc"} {
		assert.NotContainsf(t, body, `route="/api/v1/profile/`+id,
			"raw path %q leaked into a metric label", id)
	}

	// All three requests belong to a single series.
	assert.Equal(t, 1, strings.Count(body, `http_requests_total{method="GET",route="/api/v1/profile/{id}",status="200"}`))
}

func TestMetricsRecordsStatusCodes(t *testing.T) {
	t.Parallel()

	r, _ := routerWithMetrics(t)

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/boom", nil))

	body := scrape(t, r)
	assert.Contains(t, body, `status="500"`)
}

// An unrouted request must collapse into one bucket rather than minting a
// series per probed URL, which is otherwise a trivial cardinality attack.
func TestMetricsCollapsesUnmatchedRoutes(t *testing.T) {
	t.Parallel()

	r, _ := routerWithMetrics(t)

	for _, p := range []string{"/nope-1", "/nope-2", "/nope-3"} {
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, p, nil))
	}

	body := scrape(t, r)

	for _, p := range []string{"nope-1", "nope-2", "nope-3"} {
		assert.NotContains(t, body, p, "unmatched path leaked into a metric label")
	}
	assert.Contains(t, body, routeUnmatched)
}

// scrapeOpenMetrics negotiates the OpenMetrics exposition format, which is the
// only one that can carry exemplars.
func scrapeOpenMetrics(t *testing.T, h http.Handler) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	req.Header.Set("Accept", "application/openmetrics-text; version=1.0.0; charset=utf-8")

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	return w.Body.String()
}

// TestMetricsAttachesTraceExemplar covers the link between the two systems: a
// latency spike in Grafana becomes one click to the trace that caused it.
func TestMetricsAttachesTraceExemplar(t *testing.T) {
	t.Parallel()

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	t.Cleanup(func() { _ = tp.Shutdown(t.Context()) })

	m := newMetrics()
	r := chi.NewRouter()
	r.Use(tracing(tp, propagation.TraceContext{}))
	r.Use(m.middleware)
	r.Get("/ok", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/ok", nil))

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	traceId := spans[0].SpanContext.TraceID().String()

	body := scrapeOpenMetrics(t, m.handler())

	assert.Contains(t, body, "trace_id="+strconv.Quote(traceId),
		"the sampled trace id must be attached as an exemplar")
}

// TestMetricsOmitsExemplarWithoutTrace proves an unsampled or untraced request
// records latency normally rather than emitting a link that resolves to nothing.
func TestMetricsOmitsExemplarWithoutTrace(t *testing.T) {
	t.Parallel()

	r, m := routerWithMetrics(t)

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/profile/abc", nil))

	body := scrapeOpenMetrics(t, m.handler())

	assert.Contains(t, body, "http_request_duration_seconds")
	assert.NotContains(t, body, "trace_id=")
}
