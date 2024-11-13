package health

import (
	"context"
	"github.com/softika/gopherizer/database"
)

type Repository struct {
	database.Service
}

func NewRepository(db database.Service) Repository {
	return Repository{
		Service: db,
	}
}

func (r Repository) Health(ctx context.Context) map[string]string {
	return r.Service.Health(ctx)
}
