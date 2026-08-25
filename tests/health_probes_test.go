package tests

import (
	"net/http"
	"net/http/httptest"
)

// Liveness and readiness answer different questions and must be separate
// endpoints, so an orchestrator can stop traffic without restarting the pod.
func (s *E2ETestSuite) TestHealthProbes() {
	testCases := []struct {
		name     string
		path     string
		wantCode int
	}{
		{name: "liveness", path: "/health/live", wantCode: http.StatusOK},
		{name: "readiness", path: "/health/ready", wantCode: http.StatusOK},
		{name: "legacy health", path: "/health", wantCode: http.StatusOK},
	}

	for _, tc := range testCases {
		tt := tc
		s.Run(tt.name, func() {
			s.T().Parallel()

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			s.router.ServeHTTP(w, req)

			s.Equal(tt.wantCode, w.Code)

			body := w.Body.String()
			s.NotEmpty(body)

			// Pool internals must never reach an unauthenticated caller.
			for _, leak := range []string{
				"max_connections", "total_connections", "acquired_connections",
				"acquire_duration", "idle_connections",
			} {
				s.NotContains(body, leak, "%s leaked pool internals: %s", tt.path, body)
			}
		})
	}
}

// Every response must carry the ids that make logs traceable.
func (s *E2ETestSuite) TestCorrelationHeadersOnResponses() {
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)

	s.NotEmpty(w.Header().Get("X-Request-Id"))
	s.NotEmpty(w.Header().Get("X-Correlation-Id"))
}

// The metrics endpoint must be served and must not leak path parameters into
// label values.
func (s *E2ETestSuite) TestMetricsEndpoint() {
	// Generate traffic on a parameterised route first.
	get := httptest.NewRequest(http.MethodGet, "/api/v1/profile/0dd35f9a-0d20-41f1-80c2-d7993e313fb4", nil)
	s.router.ServeHTTP(httptest.NewRecorder(), get)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, req)

	s.Equal(http.StatusOK, w.Code)

	body := w.Body.String()
	s.Contains(body, "http_requests_total")
	s.Contains(body, "/api/v1/profile/{id}")
	s.NotContains(body, "0dd35f9a-0d20-41f1-80c2-d7993e313fb4",
		"a path parameter leaked into a metric label")
}
