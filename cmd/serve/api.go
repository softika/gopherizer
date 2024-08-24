package serve

import (
	"github.com/go-chi/chi/v5"
	"net/http"

	"tldw/config"
	"tldw/http/api"
)

func initApi(cfg *config.Config) *api.Router {
	router := api.NewRouter(
		cfg.App.Environment,
		cfg.Http.Auth.Secret,
	)

	r := initRepositories(cfg.Database)
	s := initServices(r)
	h := initHandlers(s)

	initRoutes(router, h)

	return router
}

func initRoutes(router *api.Router, h handlers) {
	h.healthHandler.Route(router, http.MethodGet, "/health")

	// profile
	router.Route("/profile", func(r chi.Router) {
		r.Post("/", router.CreateHttpHandlerFunc(h.createProfileHandler.Handle))
		r.Put("/", router.CreateHttpHandlerFunc(h.updateProfileHandler.Handle))
		r.Get("/{id}", router.CreateHttpHandlerFunc(h.getByIdProfileHandler.Handle))
		r.Delete("/{id}", router.CreateHttpHandlerFunc(h.deleteByIdProfileHandler.Handle))
	})
}
