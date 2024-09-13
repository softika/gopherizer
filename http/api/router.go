package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Router struct {
	chi.Router

	environment string
	secretKey   string
}

func NewRouter(environment string, secretKey string) *Router {
	r := chi.NewRouter()
	defaultMiddlewares(r)

	return &Router{
		Router:      r,
		environment: environment,
		secretKey:   secretKey,
	}
}

func defaultMiddlewares(r *chi.Mux) {
	r.Use(middleware.Logger)
	r.Use(middleware.CleanPath)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/"))
	r.Use(middleware.NoCache)
	r.Use(middleware.AllowContentEncoding("deflate", "gzip"))
}

// HandlerFunc is API generic handler func type.
type HandlerFunc[In any, Out any] func(http.ResponseWriter, *http.Request) error

// MakeHttpHandlerFunc creates http.HandlerFunc from custom HandlerFunc.
func (r *Router) MakeHttpHandlerFunc(h HandlerFunc[any, any]) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if err := h(w, req); err != nil {
			var apiError Error
			if errors.As(err, &apiError) {
				http.Error(w, apiError.Error(), apiError.Code)
				return
			}

			apiError = newError(http.StatusInternalServerError, "internal server error", err)
			http.Error(w, apiError.Error(), http.StatusInternalServerError)
		}
	}
}
