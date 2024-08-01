package serve

import "tldw/internal/services/health"

type services struct {
	healthService health.Service
}

func initServices(r repositories) services {
	return services{
		healthService: health.NewService(r.healthRepo),
	}
}
