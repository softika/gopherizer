package serve

import (
	"tldw/internal/services/health"
	"tldw/internal/services/profile"
)

type services struct {
	healthService  health.Service
	profileService profile.Service
}

func initServices(r repositories) services {
	return services{
		healthService:  health.NewService(r.healthRepo),
		profileService: profile.NewService(r.profileRepo),
	}
}
