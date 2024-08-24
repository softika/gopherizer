package serve

import (
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
}
