//go:generate mockgen -source=handler.go -destination=./mock/handler.go -package=mock
package api

import (
	"context"
	"errors"
	"net/http"

	"tldw/logger"
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

// Handle handles the request.
func (h Handler[In, Out]) Handle(w http.ResponseWriter, r *http.Request) error {
	// Map request
	in, err := h.requestMapper.Map(r)
	if err != nil {
		logger.Get().Error("failed to map request", "error", err)
		return newError(http.StatusBadRequest, err.Error(), err)
	}

	// Validate request
	if h.validator != nil {
		err = h.validator.StructCtx(r.Context(), in)
		if err != nil {
			logger.Get().Error("request validation failed", "error", err)
			return newError(http.StatusBadRequest, err.Error(), err)
		}
	}

	// Call out to service function
	out, err := h.serviceFunc(r.Context(), in)
	if err != nil {
		logger.Get().Error("service function failed", "error", err)
		return newServiceError(err)
	}

	// Map and return response
	return h.responseMapper.Map(w, out)
}

// Route registers the handler with the router.
func (h Handler[In, Out]) Route(router *Router, method, path string) {
	router.Method(method, path, h)
}

// ServeHTTP implements http.Handler interface and does error handling.
func (h Handler[In, Out]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h.Handle(w, r); err != nil {
		var apiError Error
		if errors.As(err, &apiError) {
			http.Error(w, apiError.Error(), apiError.Code)
			return
		}

		apiError = newError(http.StatusInternalServerError, "internal server error", err)
		http.Error(w, apiError.Error(), http.StatusInternalServerError)
	}
}
