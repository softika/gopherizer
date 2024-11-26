package api

import (
	"context"
	_ "embed"
	"encoding/json"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"

	"github.com/softika/slogging"
)

//go:embed docs/v1/api.yaml
var apiV1Docs []byte

//go:embed docs/swagger.html
var swaggerUI []byte

func (r *Router) initOpenApiDocs() {
	ctx := context.Background()
	loader := &openapi3.Loader{Context: ctx, IsExternalRefsAllowed: true}
	doc, err := loader.LoadFromData(apiV1Docs)
	if err != nil {
		slogging.Slogger().ErrorContext(ctx, "failed to load openapi3 document", "error", err)
		return
	}

	r.Get("/api/v1/docs", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(doc)
		if err != nil {
			slogging.Slogger().ErrorContext(ctx, "failed to encode openapi3 document", "error", err)
		}
	})

	r.Get("/docs", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, err = w.Write(swaggerUI)
		if err != nil {
			slogging.Slogger().ErrorContext(ctx, "failed to write swaggerUI", "error", err)
		}
	})
}

func (r *Router) initRoutes(h handlers) {
	h.health.Route(r, http.MethodGet, "/health")

	r.Route("/api/v1", func(c chi.Router) {
		c.Route("/profile", func(c chi.Router) {
			c.Post("/", r.HttpHandlerFunc(h.profileCreate.Handle))
			c.Put("/", r.HttpHandlerFunc(h.profileUpdate.Handle))
			c.Get("/{id}", r.HttpHandlerFunc(h.profileGet.Handle))
			c.Delete("/{id}", r.HttpHandlerFunc(h.profileDelete.Handle))
		})
	})
}
