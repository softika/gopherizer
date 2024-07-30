package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

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
