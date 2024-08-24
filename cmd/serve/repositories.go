package serve

import (
	"tldw/config"
	"tldw/database"
	"tldw/database/repositories/profile"
	"tldw/internal/services/health"
)

type repositories struct {
	healthRepo  health.Repository
	profileRepo profile.Repository
}

func initRepositories(cfg config.DatabaseConfig) repositories {
	db := database.New(cfg)
	return repositories{
		healthRepo:  db,
		profileRepo: profile.NewRepository(db),
	}
}
