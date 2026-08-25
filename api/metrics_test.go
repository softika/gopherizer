package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
