package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/softika/gopherizer/config"
)

func corsCfg(origins, methods, headers string) config.HTTPConfig {
	cfg := config.HTTPConfig{}
	cfg.Cors.Origins = origins
	cfg.Cors.Methods = methods
	cfg.Cors.Headers = headers
	return cfg
}

func TestCorsDisabledWhenNoOriginsConfigured(t *testing.T) {
	t.Parallel()

	assert.Nil(t, corsMiddleware(config.HTTPConfig{}),
		"cors must stay off unless origins are configured")
}

func TestCorsAllowsConfiguredOrigin(t *testing.T) {
	t.Parallel()

	mw := corsMiddleware(corsCfg(
		"https://app.example.com,https://admin.example.com",
		"GET,POST",
		"Content-Type",
	))
	require.NotNil(t, mw)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		name      string
		origin    string
		wantAllow string
	}{
		{"first configured origin", "https://app.example.com", "https://app.example.com"},
		{"second configured origin", "https://admin.example.com", "https://admin.example.com"},
		{"unconfigured origin", "https://evil.example.com", ""},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
			req.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.wantAllow, w.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}

// A preflight must advertise exactly the configured methods and headers.
func TestCorsPreflight(t *testing.T) {
	t.Parallel()

	mw := corsMiddleware(corsCfg("https://app.example.com", "GET,POST,DELETE", "Content-Type,X-Request-Id"))
	require.NotNil(t, mw)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("preflight must not reach the handler")
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/profile", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, "https://app.example.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
}

// Wildcard origins must never be combined with credentials: browsers reject it,
// and allowing both would expose authenticated responses to any site.
func TestCorsWildcardDoesNotAllowCredentials(t *testing.T) {
	t.Parallel()

	mw := corsMiddleware(corsCfg("*", "GET", "Content-Type"))
	require.NotNil(t, mw)

	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://anything.example.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"),
		"credentials must not be allowed alongside a wildcard origin")
}
