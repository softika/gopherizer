package api

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/softika/gopherizer/config"
	"github.com/softika/gopherizer/database"
)

// Router is the main API router.
//
// Routing and middleware stay with chi; huma sits on top and owns request
// decoding, validation, responses and the generated OpenAPI document.
type Router struct {
	chi.Router

	api         huma.API
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

	api.api = humachi.New(r, openApiConfig(cfg))

	registerOperations(api.api, api.initServices(api.initRepositories(db)))

	return api
}

// openApiConfig describes the generated document and where it is served.
func openApiConfig(cfg *config.Config) huma.Config {
	c := huma.DefaultConfig(cfg.App.Name, cfg.App.Version)

	// Served as /openapi.json and /openapi.yaml, with the browsable UI at /docs.
	c.OpenAPIPath = "/openapi"
	c.DocsPath = "/docs"

	// Package-qualified schema names, so same-named types across packages do
	// not collide in huma's single global registry.
	c.Components.Schemas = newSchemaRegistry()

	return c
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
