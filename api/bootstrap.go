package api

import (
	"github.com/go-playground/validator/v10"

	"github.com/softika/gopherizer/config"
	"github.com/softika/gopherizer/database"

	// internal services
	"github.com/softika/gopherizer/internal/health"
	"github.com/softika/gopherizer/internal/profile"

	// database repositories
	repos "github.com/softika/gopherizer/database/repositories"

	// http handler mappers
	"github.com/softika/gopherizer/api/mappers"
)

type repositories struct {
	health  health.Repository
	profile profile.Repository
}

func (r *Router) initRepositories(cfg config.DatabaseConfig) repositories {
	db := database.New(cfg)
	return repositories{
		health:  repos.NewHealthRepository(db),
		profile: repos.NewProfileRepository(db),
	}
}

type services struct {
	health  health.Service
	profile profile.Service
}

func (r *Router) initServices(s repositories) services {
	return services{
		health:  health.NewService(s.health),
		profile: profile.NewService(s.profile),
	}
}

type handlers struct {
	health Handler[health.Request, *health.Response]

	profileCreate Handler[profile.CreateRequest, *profile.Response]
	profileGet    Handler[profile.GetRequest, *profile.Response]
	profileUpdate Handler[profile.UpdateRequest, *profile.Response]
	profileDelete Handler[profile.DeleteRequest, bool]
}

func (r *Router) initHandlers(s services) handlers {
	vld := validator.New()

	healthHandler := NewHandler(
		mappers.HealthRequest{},
		mappers.HealthResponse{},
		s.health.Check,
		vld,
	)

	profileCreateHandler := NewHandler(
		mappers.CreateProfileRequest{},
		mappers.CreateProfileResponse{},
		s.profile.Create,
		vld,
	)

	profileGetHandler := NewHandler(
		mappers.GetProfileByIdRequest{},
		mappers.GetProfileResponse{},
		s.profile.GetById,
		vld,
	)

	profileUpdateHandler := NewHandler(
		mappers.UpdateProfileRequest{},
		mappers.UpdateProfileResponse{},
		s.profile.Update,
		vld,
	)

	profileDeleteHandler := NewHandler(
		mappers.DeleteProfileRequest{},
		mappers.DeleteProfileResponse{},
		s.profile.DeleteById,
		vld,
	)

	return handlers{
		health: healthHandler,

		profileCreate: profileCreateHandler,
		profileGet:    profileGetHandler,
		profileUpdate: profileUpdateHandler,
		profileDelete: profileDeleteHandler,
	}
}
