package api

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/softika/gopherizer/config"
)

// requestTimeout gives every request a deadline, returning nil when disabled.
//
// Without one, a handler is bounded only by the server's write timeout, which
// is measured in minutes -- long enough for a pool's worth of slow requests to
// hold every connection.
//
// It cancels the context rather than writing a response itself. The handler
// sees the cancellation, the query it is waiting on is abandoned, and the error
// travels back through the normal path, so the caller gets 503 from a request
// that ran out of time rather than 500 from one that broke.
//
// This is the middle of three layers: the database statement timeout fires
// first and names the slow query, this fires next, and the server's write
// timeout is the backstop.
func requestTimeout(cfg config.HTTPConfig) func(http.Handler) http.Handler {
	if cfg.RequestTimeout <= 0 {
		return nil
	}

	return middleware.Timeout(cfg.RequestTimeout)
}
