package serve

import (
	"github.com/go-playground/validator/v10"

	"tldw/http/api"

	mapperHealth "tldw/http/api/health"
	mapperProfile "tldw/http/api/profile"

	svcHealth "tldw/internal/services/health"
	svcProfile "tldw/internal/services/profile"
)

type handlers struct {
	healthHandler api.Handler[svcHealth.Request, *svcHealth.Response]

	// profile
	createProfileHandler     api.Handler[svcProfile.CreateRequest, *svcProfile.Response]
	updateProfileHandler     api.Handler[svcProfile.UpdateRequest, *svcProfile.Response]
	getByIdProfileHandler    api.Handler[string, *svcProfile.Response]
	deleteByIdProfileHandler api.Handler[string, bool]
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

		// profile
		createProfileHandler: api.NewHandler(
			mapperProfile.NewCreateRequestMapper(),
			mapperProfile.NewCreateResponseMapper(),
			s.profileService.Create,
			vld,
		),
		updateProfileHandler: api.NewHandler(
			mapperProfile.NewUpdateRequestMapper(),
			mapperProfile.NewUpdateResponseMapper(),
			s.profileService.Update,
			vld,
		),
		getByIdProfileHandler: api.NewHandler(
			mapperProfile.NewGetByIdRequestMapper(),
			mapperProfile.NewGetByIdResponseMapper(),
			s.profileService.GetById,
			vld,
		),
		deleteByIdProfileHandler: api.NewHandler(
			mapperProfile.NewDeleteByIdRequestMapper(),
			mapperProfile.NewDeleteByIdResponseMapper(),
			s.profileService.DeleteById,
			vld,
		),
	}
}
