package repositories

import (
	"context"

	"github.com/softika/gopherizer/database"
)

type HealthRepository struct {
	database.Service
}

func NewHealthRepository(db database.Service) HealthRepository {
	return HealthRepository{
		Service: db,
	}
}

func (r HealthRepository) Health(ctx context.Context) map[string]string {
	return r.Service.Health(ctx)
}
