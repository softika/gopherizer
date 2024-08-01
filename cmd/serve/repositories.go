package serve

import (
	"tldw/config"
	"tldw/database"
	"tldw/internal/services/health"
)

type repositories struct {
	healthRepo health.Repository
}

func initRepositories(cfg config.DatabaseConfig) repositories {
	return repositories{
		healthRepo: database.New(cfg),
	}
}
