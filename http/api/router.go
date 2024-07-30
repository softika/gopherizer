package api

import (
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
