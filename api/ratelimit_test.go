package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/softika/gopherizer/config"
)

func TestRateLimiterDisabledWhenUnconfigured(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  config.HTTPConfig
	}{
		{name: "zero requests", cfg: config.HTTPConfig{}},
		{
			name: "zero window",
			cfg: func() config.HTTPConfig {
				c := config.HTTPConfig{}
				c.RateLimit.Requests = 10
				return c
			}(),
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Nil(t, rateLimiter(tt.cfg), "rate limiting must stay off unless fully configured")
		})
	}
}

func TestRateLimiterRejectsOverLimit(t *testing.T) {
	t.Parallel()

	cfg := config.HTTPConfig{}
	cfg.RateLimit.Requests = 3
	cfg.RateLimit.Window = time.Minute

	limiter := rateLimiter(cfg)
	require.NotNil(t, limiter, "rate limiting must be active when configured")

	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	call := func() int {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/profile/1", nil)
		req.RemoteAddr = "192.0.2.10:1234"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w.Code
	}

	for i := 1; i <= 3; i++ {
		assert.Equal(t, http.StatusOK, call(), "request %d is within the limit", i)
	}

	assert.Equal(t, http.StatusTooManyRequests, call(), "the 4th request must be rejected")
}

// Limiting is per client, so one noisy caller must not lock everyone else out.
func TestRateLimiterIsPerClient(t *testing.T) {
	t.Parallel()

	cfg := config.HTTPConfig{}
	cfg.RateLimit.Requests = 1
	cfg.RateLimit.Window = time.Minute

	handler := rateLimiter(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	call := func(addr string) int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = addr
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w.Code
	}

	assert.Equal(t, http.StatusOK, call("192.0.2.20:1111"))
	assert.Equal(t, http.StatusTooManyRequests, call("192.0.2.20:1111"))
	assert.Equal(t, http.StatusOK, call("192.0.2.21:2222"), "a different client must be unaffected")
}

// The limiter must key on the socket address, not a client-supplied header,
// otherwise a caller can bypass it by spoofing X-Forwarded-For.
func TestRateLimiterIgnoresSpoofedForwardedHeader(t *testing.T) {
	t.Parallel()

	cfg := config.HTTPConfig{}
	cfg.RateLimit.Requests = 1
	cfg.RateLimit.Window = time.Minute

	handler := rateLimiter(cfg)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	call := func(forwarded string) int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.0.2.30:9999"
		req.Header.Set("X-Forwarded-For", forwarded)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w.Code
	}

	assert.Equal(t, http.StatusOK, call("10.0.0.1"))
	assert.Equal(t, http.StatusTooManyRequests, call("10.0.0.2"),
		"rotating X-Forwarded-For must not reset the limit")
}
