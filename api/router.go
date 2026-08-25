package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/softika/gopherizer/config"
	"github.com/softika/gopherizer/database"
)

// Router is the main API router.
// It is a wrapper around chi.Router with some additional functionality.
// Chi router can be replaced with any other router that implements net/http.
type Router struct {
	chi.Router

	environment string
}

// NewRouter builds the API router around an already-connected database.
// The pool is injected rather than created here so its lifetime is owned by the
// caller, which is what allows a clean shutdown.
func NewRouter(cfg *config.Config, db database.Service) *Router {
	r := chi.NewRouter()
	defaultMiddlewares(r, cfg.Http)

	api := &Router{
		Router:      r,
		environment: cfg.App.Environment,
	}

	s := api.initServices(api.initRepositories(db))
	h := api.initHandlers(s)

	api.initRoutes(h)
	api.initOpenApiDocs()

	return api
}

// metricsPath returns the configured scrape path, falling back to the
// conventional default when unset.
func metricsPath(cfg config.HTTPConfig) string {
	if cfg.Metrics.Path == "" {
		return "/metrics"
	}
	return cfg.Metrics.Path
}

func defaultMiddlewares(r *chi.Mux, cfg config.HTTPConfig) {
	// Correlation must run first so every later log line carries the ids.
	r.Use(correlation)
	r.Use(middleware.Logger)
	r.Use(middleware.CleanPath)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/"))
	r.Use(middleware.NoCache)
	r.Use(middleware.AllowContentEncoding("deflate", "gzip"))

	if c := corsMiddleware(cfg); c != nil {
		r.Use(c)
	}

	if cfg.Metrics.Enabled {
		m := newMetrics()
		r.Use(m.middleware)
		r.Handle(metricsPath(cfg), m.handler())
	}

	if limiter := rateLimiter(cfg); limiter != nil {
		// The resolver must run before the limiter so the bucket key is correct.
		r.Use(clientIPResolver(cfg))
		r.Use(limiter)
	}
}

// HandlerFunc is API generic handler func type.
type HandlerFunc func(http.ResponseWriter, *http.Request) error

// HttpHandlerFunc creates http.HandlerFunc from custom HandlerFunc.
// It handles API errors and returns them as HTTP errors.
func (r *Router) HttpHandlerFunc(h HandlerFunc) http.HandlerFunc {
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
