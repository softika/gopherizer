package api

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/softika/gopherizer/config"
)

// stubDB satisfies database.Service for construction-time tests. NewRouter
// wires repositories but issues no queries, so the pool is never dereferenced.
type stubDB struct{}

func (stubDB) Health(context.Context) map[string]string { return map[string]string{"status": "up"} }
func (stubDB) Close() error                             { return nil }
func (stubDB) DB() *sql.DB                              { return nil }
func (stubDB) Pool() *pgxpool.Pool                      { return nil }

// TestNewRouterWithShippedDefaults builds the router from default.toml.
//
// Config assembled by hand in a test can silently disable features and hide a
// wiring fault: chi panics if any middleware is registered after a route, and
// that only happens when metrics and rate limiting are both enabled, which is
// precisely what the shipped defaults do.
func TestNewRouterWithShippedDefaults(t *testing.T) {
	cfg, err := config.New()
	require.NoError(t, err)

	require.True(t, cfg.Http.Metrics.Enabled, "defaults must enable metrics for this to be meaningful")
	require.Positive(t, cfg.Http.RateLimit.Requests, "defaults must enable rate limiting")

	var router *Router
	require.NotPanics(t, func() {
		router = NewRouter(cfg, stubDB{})
	}, "the router must build with the configuration the template ships")

	require.NotNil(t, router)

	// The stack must actually serve, not merely construct.
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Metrics must be reachable at the configured path.
	req = httptest.NewRequest(http.MethodGet, metricsPath(cfg.Http), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code, "metrics endpoint must be registered")

	// Rate limiting must be active, so the ids it keys on are resolved.
	req = httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	assert.NotEmpty(t, w.Header().Get(requestIdHeader), "correlation middleware must run")
}
