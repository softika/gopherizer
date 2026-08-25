package api

import (
	"net/http"

	"github.com/go-playground/validator/v10"

	"github.com/softika/gopherizer/database"

	// internal services
	"github.com/softika/gopherizer/internal/health"
	"github.com/softika/gopherizer/internal/profile"

	// database repositories
	repos "github.com/softika/gopherizer/database/repositories"
)

type repositories struct {
	health  health.Repository
	profile profile.Repository
}

func (r *Router) initRepositories(db database.Service) repositories {
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
	health     Handler[health.Request, *health.Response]
	healthLive Handler[health.Request, *health.Response]

	profileCreate Handler[profile.CreateRequest, *profile.Response]
	profileGet    Handler[profile.GetRequest, *profile.Response]
	profileUpdate Handler[profile.UpdateRequest, *profile.Response]
	profileDelete Handler[profile.DeleteRequest, bool]
}

func (r *Router) initHandlers(s services) handlers {
	vld := validator.New()

	// Each endpoint is a decoder, an encoder and the service call to make.
	// The reusable decoders and encoders live in codec.go.
	byId := func(id string) profile.GetRequest { return profile.GetRequest{Id: id} }
	deleteById := func(id string) profile.DeleteRequest { return profile.DeleteRequest{Id: id} }

	return handlers{
		health: NewHandler(
			Static(health.Request{Status: "OK"}),
			JSON[*health.Response](http.StatusOK),
			s.health.Check,
			vld,
		),
		healthLive: NewHandler(
			Static(health.Request{Status: "OK"}),
			JSON[*health.Response](http.StatusOK),
			s.health.Live,
			vld,
		),
		profileCreate: NewHandler(
			JSONBody[profile.CreateRequest],
			JSON[*profile.Response](http.StatusCreated),
			s.profile.Create,
			vld,
		),
		profileGet: NewHandler(
			PathParam("id", byId),
			JSON[*profile.Response](http.StatusOK),
			s.profile.GetById,
			vld,
		),
		profileUpdate: NewHandler(
			JSONBody[profile.UpdateRequest],
			JSON[*profile.Response](http.StatusOK),
			s.profile.Update,
			vld,
		),
		profileDelete: NewHandler(
			PathParam("id", deleteById),
			NoContent[bool],
			s.profile.DeleteById,
			vld,
		),
	}
}
