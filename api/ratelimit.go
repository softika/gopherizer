package api

import (
	"net/http"

	"github.com/go-chi/httprate"

	"github.com/softika/gopherizer/config"
)

// rateLimiter builds the per-client rate limiting middleware, or nil when the
// limiter is not fully configured.
//
// Requests are bucketed by the address resolved by clientIPResolver, so the key
// follows the deployment's declared trust model rather than assuming one.
func rateLimiter(cfg config.HTTPConfig) func(http.Handler) http.Handler {
	if cfg.RateLimit.Requests <= 0 || cfg.RateLimit.Window <= 0 {
		return nil
	}

	return httprate.LimitBy(cfg.RateLimit.Requests, cfg.RateLimit.Window, clientIPKey)
}
