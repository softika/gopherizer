package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (r *Router) initRoutes(h handlers) {
	h.health.Route(r, http.MethodGet, "/health")

	r.Route("/api/v1", func(c chi.Router) {

		c.Route("/account", func(c chi.Router) {
			c.Post("/register", r.MakeHttpHandlerFunc(h.accountRegister.Handle))
			c.Post("/login", r.MakeHttpHandlerFunc(h.accountLogin.Handle))
			c.Put("/change-password", r.MakeHttpHandlerFunc(h.accountChangePassword.Handle))
		})

		c.Route("/profile", func(c chi.Router) {
			c.Post("/", r.MakeHttpHandlerFunc(h.profileCreate.Handle))
			c.Put("/", r.MakeHttpHandlerFunc(h.profileUpdate.Handle))
			c.Get("/{id}", r.MakeHttpHandlerFunc(h.profileGet.Handle))
			c.Delete("/{id}", r.MakeHttpHandlerFunc(h.profileDelete.Handle))
		})

	})

}
