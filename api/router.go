package api

import (
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel"

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

	// Every middleware first; routes only afterwards.
	m := defaultMiddlewares(r, cfg, db)

	if m != nil {
		r.Handle(metricsPath(cfg.Http), m.handler())
	}

	api := &Router{
		Router:      r,
		environment: cfg.App.Environment,
	}

	api.api = humachi.New(r, openApiConfig(cfg))

	registerOperations(api.api, api.initServices(api.initRepositories(db)), cfg.Http.MaxBodyBytes)

	return api
}

// openApiConfig describes the generated document and where it is served.
func openApiConfig(cfg *config.Config) huma.Config {
	c := huma.DefaultConfig(cfg.App.Name, cfg.App.Version)

	// Served as /openapi.json and /openapi.yaml, with the browsable UI at /docs.
	c.OpenAPIPath = "/openapi"
	c.DocsPath = "/docs"

	// huma defaults to Stoplight Elements, which renders a "powered by
	// Stoplight" credit. SwaggerUI carries no third-party attribution and, with
	// the default BaseLayout huma uses, no vendor topbar either.
	c.DocsRenderer = huma.DocsRendererSwaggerUI

	// Package-qualified schema names, so same-named types across packages do
	// not collide in huma's single global registry.
	c.Components.Schemas = newSchemaRegistry()

	// DefaultConfig's only create hook installs a schema-link transformer that
	// stamps a "$schema" field into every response body and a matching Link
	// header. Both embed the server's own host, putting deployment detail into
	// payloads, so the hook is dropped. The OpenAPI document is unaffected.
	c.CreateHooks = nil

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

// defaultMiddlewares registers the middleware stack and returns the metrics
// collectors when they are enabled, so the caller can mount the scrape endpoint.
//
// chi panics if Use is called after any route is registered, so this function
// must not register routes: every r.Use in the process has to happen before the
// first r.Handle.
func defaultMiddlewares(r *chi.Mux, cfg *config.Config, db database.Service) *metrics {
	// Tracing runs first so the span is on the context that every later
	// middleware sees, which is what lets the access log carry a trace id.
	if cfg.Tracing.Enabled {
		r.Use(tracing(otel.GetTracerProvider(), otel.GetTextMapPropagator()))
	}

	// Correlation runs next so every later log line carries the ids,
	// including the access log line registered immediately after it.
	r.Use(correlation)
	r.Use(accessLogger(cfg.Http, slog.Default()))
	r.Use(middleware.CleanPath)

	// Structured, rather than chi's Recoverer, which writes the stack to
	// stderr as plain text with no correlation id.
	r.Use(recoverer(slog.Default()))

	// Inside the recoverer, so a panic raised while unwinding a timed-out
	// request is still caught and reported.
	if timeout := requestTimeout(cfg.Http); timeout != nil {
		r.Use(timeout)
	}

	r.Use(middleware.Heartbeat("/"))
	r.Use(middleware.NoCache)
	r.Use(middleware.AllowContentEncoding("deflate", "gzip"))

	// Outside huma, so the cap covers every route the router serves rather
	// than only the registered operations.
	if limit := bodyLimit(cfg.Http); limit != nil {
		r.Use(limit)
	}

	if c := corsMiddleware(cfg.Http); c != nil {
		r.Use(c)
	}

	var m *metrics
	if cfg.Http.Metrics.Enabled {
		m = newMetrics(
			withBuildInfo(cfg.App),
			withPoolStats(poolStatsFrom(db)),
		)
		r.Use(m.middleware)
	}

	if limiter := rateLimiter(cfg.Http); limiter != nil {
		// The resolver must run before the limiter so the bucket key is correct.
		r.Use(clientIPResolver(cfg.Http))
		r.Use(limiter)
	}

	return m
}
