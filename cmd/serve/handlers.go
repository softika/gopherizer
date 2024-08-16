package serve

import (
	"github.com/go-playground/validator/v10"

	"tldw/http/api"
	mapperHealth "tldw/http/api/health"
	svcHealth "tldw/internal/services/health"
)

type handlers struct {
	healthHandler api.Handler[svcHealth.Request, *svcHealth.Response]
}

func initHandlers(s services) handlers {
	// init validator
	vld := validator.New()

	// init handlers
	return handlers{
		healthHandler: api.NewHandler(
			mapperHealth.NewRequestMapper(),
			mapperHealth.NewResponseMapper(),
			s.healthService.Check,
			vld,
		),
	}
}
