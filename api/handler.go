//go:generate mockgen -source=handler.go -destination=./mock/handler.go -package=mock
package api

import (
	"context"
	"net/http"

	"github.com/softika/slogging"
)

// ServiceFunc is a generic service function type called in a handler.
type ServiceFunc[In any, Out any] func(context.Context, In) (Out, error)

// RequestMapper is generic interfaces for mapping request objects.
type RequestMapper[In any] interface {
	Map(*http.Request) (In, error)
}

// ResponseMapper is generic interfaces for mapping response objects.
type ResponseMapper[Out any] interface {
	Map(http.ResponseWriter, Out) error
}

type Validator interface {
	StructCtx(ctx context.Context, s interface{}) (err error)
}

// Handler is a generic handler type.
type Handler[In any, Out any] struct {
	serviceFunc    ServiceFunc[In, Out]
	requestMapper  RequestMapper[In]
	responseMapper ResponseMapper[Out]
	validator      Validator
}

// NewHandler creates a new handler.
func NewHandler[In any, Out any](
	reqMapper RequestMapper[In],
	resMapper ResponseMapper[Out],
	svcFunc ServiceFunc[In, Out],
	vld Validator,
) Handler[In, Out] {
	return Handler[In, Out]{
		requestMapper:  reqMapper,
		responseMapper: resMapper,
		serviceFunc:    svcFunc,
		validator:      vld,
	}
}

// Handle handles the http request.
func (h Handler[In, Out]) Handle(w http.ResponseWriter, r *http.Request) error {
	logger := slogging.Slogger()

	// map request
	in, err := h.requestMapper.Map(r)
	if err != nil {
		logger.ErrorContext(r.Context(), "failed to map request", "error", err)
		return newError(http.StatusBadRequest, err.Error(), err)
	}

	// validate request
	if h.validator != nil {
		err = h.validator.StructCtx(r.Context(), in)
		if err != nil {
			logger.ErrorContext(r.Context(), "request validation failed", "error", err)
			return newError(http.StatusBadRequest, err.Error(), err)
		}
	}

	// call out to service function
	out, err := h.serviceFunc(r.Context(), in)
	if err != nil {
		logger.ErrorContext(r.Context(), "service function failed", "error", err)
		return newServiceError(err)
	}

	// map and return response
	return h.responseMapper.Map(w, out)
}

// Route registers the handler with the router.
func (h Handler[In, Out]) Route(router *Router, method, path string) {
	router.Method(method, path, router.MakeHttpHandlerFunc(h.Handle))
}
