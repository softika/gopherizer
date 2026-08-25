package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/softika/gopherizer/config"
)

// TestHttpServerTimeouts guards the server against slow-client attacks.
// ReadHeaderTimeout is the one that matters: without it a client can hold a
// connection open indefinitely by dribbling headers (Slowloris, gosec G112).
func TestHttpServerTimeouts(t *testing.T) {
	t.Parallel()

	cfg := config.HTTPConfig{
		Host:              "127.0.0.1",
		Port:              "8080",
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      31 * time.Second,
		IdleTimeout:       32 * time.Second,
	}

	srv := NewServer(cfg).httpServer(http.NotFoundHandler())

	assert.Equal(t, "127.0.0.1:8080", srv.Addr)
	assert.Equal(t, 30*time.Second, srv.ReadTimeout)
	assert.Equal(t, 31*time.Second, srv.WriteTimeout)

	assert.NotZero(t, srv.ReadHeaderTimeout, "ReadHeaderTimeout must be set (Slowloris)")
	assert.Equal(t, 5*time.Second, srv.ReadHeaderTimeout)

	assert.NotZero(t, srv.IdleTimeout, "IdleTimeout is configured but was never applied")
	assert.Equal(t, 32*time.Second, srv.IdleTimeout)
}

// A missing ReadHeaderTimeout must fall back to something bounded rather than
// silently leaving the server unprotected.
func TestHttpServerReadHeaderTimeoutFallback(t *testing.T) {
	t.Parallel()

	srv := NewServer(config.HTTPConfig{Port: "8080"}).httpServer(http.NotFoundHandler())

	assert.NotZero(t, srv.ReadHeaderTimeout,
		"an unset ReadHeaderTimeout must fall back to a bounded default")
}
