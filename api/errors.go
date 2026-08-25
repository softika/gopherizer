package api

import (
	"context"
	"log/slog"

	"github.com/danielgtaylor/huma/v2"

	"github.com/softika/gopherizer/pkg/errorx"
)

// clientMessages are the client-facing renderings of each domain error type.
//
// They are deliberately generic. The underlying error frequently wraps a driver
// failure carrying table names, hostnames and SQLSTATE codes, none of which may
// reach a caller.
var clientMessages = map[errorx.ErrorType]string{
	errorx.ErrInvalidInput: "invalid request",
	errorx.ErrUnauthorized: "unauthorized",
	errorx.ErrForbidden:    "forbidden",
	errorx.ErrNotFound:     "not found",
	errorx.ErrUnavailable:  "service unavailable",
	errorx.ErrInternal:     "internal server error",
}

// statusError builds the huma error for a domain error type.
func statusError(t errorx.ErrorType) huma.StatusError {
	msg := clientMessages[t]

	switch t {
	case errorx.ErrInvalidInput:
		return huma.Error400BadRequest(msg)
	case errorx.ErrUnauthorized:
		return huma.Error401Unauthorized(msg)
	case errorx.ErrForbidden:
		return huma.Error403Forbidden(msg)
	case errorx.ErrNotFound:
		return huma.Error404NotFound(msg)
	case errorx.ErrUnavailable:
		return huma.Error503ServiceUnavailable(msg)
	case errorx.ErrInternal:
		return huma.Error500InternalServerError(clientMessages[errorx.ErrInternal])
	default:
		return huma.Error500InternalServerError(clientMessages[errorx.ErrInternal])
	}
}

// apiError logs the full failure and returns the safe response to the caller.
//
// The detail stays server-side, correlated by the request id that the
// correlation middleware put on the context.
func apiError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	errType := errorx.TypeOf(err)

	slog.ErrorContext(ctx, "request failed",
		"error", err,
		"type", errType,
	)

	return statusError(errType)
}
