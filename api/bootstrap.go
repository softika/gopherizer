package api

import (
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
