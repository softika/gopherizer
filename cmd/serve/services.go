package serve

import "tldw/internal/services/health"

type services struct {
	healthService health.Service
}

func initServices() services {
	return services{
		healthService: health.NewService(),
	}
}
