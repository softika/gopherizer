package api

import (
	"net/http"

	"github.com/softika/gopherizer/config"
)

// bodyLimit caps the size of a request body, returning nil when disabled.
//
// huma already applies a 1 MiB default to any operation that reads a body, so
// this is not the difference between bounded and unbounded for those routes.
// What it adds is that the limit becomes explicit and configurable in one
// place, and that it covers routes huma never sees -- the heartbeat, the
// metrics endpoint, and anything mounted on the chi router directly.
//
// The registered operations are given the same configured value, so the two
// limits cannot drift apart. See registerProfile.
func bodyLimit(cfg config.HTTPConfig) func(http.Handler) http.Handler {
	if cfg.MaxBodyBytes <= 0 {
		return nil
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// MaxBytesReader also tells the server to stop reading, so a client
			// streaming an endless body is disconnected rather than tolerated.
			r.Body = http.MaxBytesReader(w, r.Body, cfg.MaxBodyBytes)

			next.ServeHTTP(w, r)
		})
	}
}
