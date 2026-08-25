package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/softika/gopherizer/config"
)

// rateLimitCfg builds a config allowing exactly one request per client.
func rateLimitCfg(from string, proxies int, header string) config.HTTPConfig {
	cfg := config.HTTPConfig{}
	cfg.RateLimit.Requests = 1
	cfg.RateLimit.Window = time.Minute
	cfg.ClientIP.From = from
	cfg.ClientIP.TrustedProxies = proxies
	cfg.ClientIP.TrustedHeader = header
	return cfg
}

// limitedHandler wires the resolver and limiter exactly as the router does.
func limitedHandler(cfg config.HTTPConfig) http.Handler {
	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h = rateLimiter(cfg)(h)
	return clientIPResolver(cfg)(h)
}

func TestClientIPTrustModels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  config.HTTPConfig
		// two requests from the same socket but different forged headers
		headerName string
		first      string
		second     string
		wantSecond int
		reason     string
	}{
		{
			name:       "remote_addr ignores forged XFF",
			cfg:        rateLimitCfg(clientIPFromRemoteAddr, 0, ""),
			headerName: "X-Forwarded-For",
			first:      "10.0.0.1",
			second:     "10.0.0.2",
			wantSecond: http.StatusTooManyRequests,
			reason:     "rotating X-Forwarded-For must not reset the limit",
		},
		{
			name:       "header mode with no header configured falls back safely",
			cfg:        rateLimitCfg(clientIPFromHeader, 0, ""),
			headerName: "X-Forwarded-For",
			first:      "10.0.0.1",
			second:     "10.0.0.2",
			wantSecond: http.StatusTooManyRequests,
			reason:     "a misconfigured header source must not become spoofable",
		},
		{
			name:       "xff behind one trusted proxy distinguishes clients",
			cfg:        rateLimitCfg(clientIPFromXFF, 1, ""),
			headerName: "X-Forwarded-For",
			first:      "10.0.0.1",
			second:     "10.0.0.2",
			wantSecond: http.StatusOK,
			reason:     "distinct clients behind a trusted proxy must not share a bucket",
		},
		{
			name:       "trusted header distinguishes clients",
			cfg:        rateLimitCfg(clientIPFromHeader, 0, "Cf-Connecting-Ip"),
			headerName: "Cf-Connecting-Ip",
			first:      "10.0.0.1",
			second:     "10.0.0.2",
			wantSecond: http.StatusOK,
			reason:     "a proxy-set header must identify the caller when trusted",
		},
	}

	for _, tc := range tests {
		tt := tc
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := limitedHandler(tt.cfg)

			call := func(v string) int {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.RemoteAddr = "192.0.2.100:5555"
				req.Header.Set(tt.headerName, v)
				w := httptest.NewRecorder()
				h.ServeHTTP(w, req)
				return w.Code
			}

			assert.Equal(t, http.StatusOK, call(tt.first), "first request must pass")
			assert.Equal(t, tt.wantSecond, call(tt.second), tt.reason)
		})
	}
}

// Behind a proxy every caller shares the socket address, so keying on it would
// put unrelated clients in one bucket. This is the case the trust model exists
// to prevent.
func TestClientIPBehindProxyDoesNotShareBucket(t *testing.T) {
	t.Parallel()

	h := limitedHandler(rateLimitCfg(clientIPFromXFF, 1, ""))

	call := func(clientIP string) int {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.0.2.200:443" // the load balancer, identical for all
		req.Header.Set("X-Forwarded-For", clientIP)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w.Code
	}

	assert.Equal(t, http.StatusOK, call("203.0.113.1"))
	assert.Equal(t, http.StatusOK, call("203.0.113.2"), "a second client must not be blocked by the first")
	assert.Equal(t, http.StatusTooManyRequests, call("203.0.113.1"), "the first client is still limited")
}
