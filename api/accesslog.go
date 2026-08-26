package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/softika/gopherizer/config"
)

// statusImplicitOK is the status Go writes when a handler returns without ever
// calling WriteHeader. The wrapped writer reports 0 in that case.
const statusImplicitOK = http.StatusOK

// accessLogger records one structured line per request.
//
// It replaces chi's middleware.Logger, which writes coloured plain text through
// the standard library logger. That put two different formats on stdout -- half
// of which no log aggregator could parse -- and, more importantly, left the one
// line that records a request's status and latency with no correlation id, so
// it could not be joined to the application logs for the same request.
//
// The logger is injected rather than taken from slog.Default so the output can
// be asserted on in a test.
func accessLogger(cfg config.HTTPConfig, logger *slog.Logger) func(http.Handler) http.Handler {
	// Prometheus scrapes on a short interval, and those requests would
	// otherwise bury the logs this middleware exists to produce.
	skip := metricsPath(cfg)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == skip {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			elapsed := time.Since(start)

			status := ww.Status()
			if status == 0 {
				status = statusImplicitOK
			}

			// The request context carries the ids correlation put there, so
			// this runs through LogAttrs with a context rather than slog.Info.
			logger.LogAttrs(r.Context(), levelForStatus(status), "http request",
				slog.String("method", r.Method),
				// The matched pattern, never the raw path: see routePattern.
				slog.String("route", routePattern(r)),
				slog.String("path", r.URL.Path),
				slog.Int("status", status),
				slog.Int("bytes", ww.BytesWritten()),
				slog.Float64("duration_ms", float64(elapsed.Nanoseconds())/float64(time.Millisecond)),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
			)
		})
	}
}

// levelForStatus maps a response status onto a log level, so a failing endpoint
// is findable without knowing to filter on a status field.
func levelForStatus(status int) slog.Level {
	switch {
	case status >= http.StatusInternalServerError:
		return slog.LevelError
	case status >= http.StatusBadRequest:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}
