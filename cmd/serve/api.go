package serve

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"tldw/config"
	"tldw/http/api"
)

func initApi(cfg *config.Config) *api.Router {
	router := api.NewRouter(
		cfg.App.Environment,
		cfg.Auth.Secret,
	)

	r := initRepositories(cfg.Database)
	s := initServices(r)
	h := initHandlers(s)

	initRoutes(router, h)

	return router
}

func initRoutes(r *api.Router, h handlers) {
	h.healthHandler.Route(r, http.MethodGet, "/health")

	// profile
	r.Route("/profile", func(c chi.Router) {
		c.Post("/", r.MakeHttpHandlerFunc(h.createProfileHandler.Handle))
		c.Put("/", r.MakeHttpHandlerFunc(h.updateProfileHandler.Handle))
		c.Get("/{id}", r.MakeHttpHandlerFunc(h.getByIdProfileHandler.Handle))
		c.Delete("/{id}", r.MakeHttpHandlerFunc(h.deleteByIdProfileHandler.Handle))
	})
}
