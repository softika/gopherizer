package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/danielgtaylor/huma/v2"
)

// problemContentType is the media type huma uses for every error it writes.
const problemContentType = "application/problem+json"

// recoverer turns a panic into a logged failure and a 500.
//
// It replaces chi's middleware.Recoverer, which writes the stack straight to
// os.Stderr as plain text. That left the single most important record in the
// system in a format no aggregator could parse and with no request id, so a
// panic could not be tied back to the request that caused it -- the same defect
// the access log had, on the line where it matters most.
//
// The logger is injected so a test can assert on the record.
func recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// The context is passed in rather than read inside the closure:
			// r is never reassigned here, so the value is identical, and it
			// keeps the context flowing through an explicit parameter.
			defer func(ctx context.Context) {
				if rvr := recover(); rvr != nil {
					reportPanic(ctx, logger, w, r, rvr)
				}
			}(r.Context())

			next.ServeHTTP(w, r)
		})
	}
}

// reportPanic records the failure and answers the caller.
//
// The stack is captured here rather than in the caller so it is still the
// panicking one.
func reportPanic(
	ctx context.Context, logger *slog.Logger, w http.ResponseWriter, r *http.Request, rvr any,
) {
	// ErrAbortHandler is the standard library's way of saying "drop this
	// connection deliberately". Reporting it as a fault would be wrong, so it
	// is passed straight back up.
	if err, ok := rvr.(error); ok && errors.Is(err, http.ErrAbortHandler) {
		panic(rvr)
	}

	logger.LogAttrs(ctx, slog.LevelError, "panic recovered",
		slog.String("panic", fmt.Sprint(rvr)),
		slog.String("stack", string(debug.Stack())),
		slog.String("method", r.Method),
		slog.String("route", routePattern(r)),
		slog.String("path", r.URL.Path),
	)

	writeInternalError(ctx, logger, w)
}

// writeInternalError answers with the same shape huma uses for its own errors,
// so a client parses one error format rather than two.
//
// Nothing about the panic reaches the caller: a stack names internal paths and
// types, and sometimes values.
func writeInternalError(ctx context.Context, logger *slog.Logger, w http.ResponseWriter) {
	body, err := json.Marshal(huma.ErrorModel{
		Title:  http.StatusText(http.StatusInternalServerError),
		Status: http.StatusInternalServerError,
	})
	if err != nil {
		// Should be unreachable for a struct of a string and an int, but a
		// silent failure here would leave the caller with an empty 200.
		logger.ErrorContext(ctx, "failed to encode error response", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", problemContentType)
	w.WriteHeader(http.StatusInternalServerError)

	if _, err = w.Write(body); err != nil {
		// The response has already begun, so this cannot be reported to the
		// caller; the panic itself is logged above regardless.
		logger.ErrorContext(ctx, "failed to write error response", "error", err)
	}
}
