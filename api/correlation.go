package api

import (
	"context"
	"net/http"
	"uuid"

	"github.com/softika/slogging"
)

const (
	requestIdHeader     = "X-Request-Id"
	correlationIdHeader = "X-Correlation-Id"

	// maxIdLength bounds an inbound id so a caller cannot bloat every log line.
	maxIdLength = 128
)

// correlation puts a request id and a correlation id on the request context.
//
// slogging's handler reads both keys off the context and stamps them onto every
// record, so this middleware is what makes logs traceable across layers. Both
// are echoed back so callers can tie their own logs to ours.
//
// The two ids differ in scope: the correlation id spans systems and is
// preserved when a caller supplies one, while the request id identifies this
// hop alone and is always freshly generated.
func correlation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// v7 embeds a timestamp, so ids sort chronologically. That makes a log
		// aggregator's range queries cheap and keeps any index on the id local.
		requestId := uuid.NewV7().String()

		correlationId := safeId(r.Header.Get(correlationIdHeader))
		if correlationId == "" {
			// First hop in the chain: seed the trace from this request.
			correlationId = requestId
		}

		ctx := context.WithValue(r.Context(), slogging.RequestIdKey, requestId)
		ctx = context.WithValue(ctx, slogging.CorrelationIdKey, correlationId)

		w.Header().Set(requestIdHeader, requestId)
		w.Header().Set(correlationIdHeader, correlationId)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// safeId accepts an inbound id only when it is a bounded, printable token.
//
// These values are written into logs, so anything else is discarded rather than
// sanitized: a caller must not be able to forge log entries by smuggling
// newlines or quotes through a header.
func safeId(v string) string {
	if v == "" || len(v) > maxIdLength {
		return ""
	}

	for _, c := range v {
		switch {
		case c >= 'a' && c <= 'z',
			c >= 'A' && c <= 'Z',
			c >= '0' && c <= '9',
			c == '-', c == '_', c == '.':
		default:
			return ""
		}
	}

	return v
}
