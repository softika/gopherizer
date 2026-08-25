package api

import (
	"context"
	"log/slog"
	"net/http"
)

// ServiceFunc is a generic service function type called in a handler.
type ServiceFunc[In any, Out any] func(context.Context, In) (Out, error)

type Validator interface {
	StructCtx(ctx context.Context, s interface{}) (err error)
}

// Handler is a generic handler type.
type Handler[In any, Out any] struct {
	serviceFunc ServiceFunc[In, Out]
	decode      Decoder[In]
	encode      Encoder[Out]
	validator   Validator
}

// NewHandler creates new handler.
func NewHandler[In any, Out any](
	decode Decoder[In],
	encode Encoder[Out],
	svcFunc ServiceFunc[In, Out],
	vld Validator,
) Handler[In, Out] {
	return Handler[In, Out]{
		decode:      decode,
		encode:      encode,
		serviceFunc: svcFunc,
		validator:   vld,
	}
}

// Handle runs the request pipeline: decode, validate, call the service, encode.
//
// No endpoint needs its own handler. Compose one from a Decoder and an Encoder
// (see codec.go for the reusable ones) plus the service function to call. Each
// stage stops the pipeline on failure, so a later stage never runs on input an
// earlier one rejected.
func (h Handler[In, Out]) Handle(w http.ResponseWriter, r *http.Request) error {
	// decode request
	in, err := h.decode(r)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to decode request", "error", err)
		return newError(http.StatusBadRequest, err.Error(), err)
	}

	// validate request
	if h.validator != nil {
		err = h.validator.StructCtx(r.Context(), in)
		if err != nil {
			slog.ErrorContext(r.Context(), "request validation failed", "error", err)
			return newError(http.StatusBadRequest, err.Error(), err)
		}
	}

	// call out to service function
	out, err := h.serviceFunc(r.Context(), in)
	if err != nil {
		slog.ErrorContext(r.Context(), "service function failed", "error", err)
		return newServiceError(err)
	}

	// encode and return response
	return h.encode(w, out)
}
