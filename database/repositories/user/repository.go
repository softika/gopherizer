package user

import (
	"context"
	_ "embed"

	"github.com/oklog/ulid/v2"

	"tldw/database"
	"tldw/internal/model"
)

var (
	//go:embed sql/get_by_id.sql
	getById string
	//go:embed sql/get_by_email.sql
	getByEmail string
)

type Repository struct {
	dbService database.Service
}

func NewRepository(dbService database.Service) Repository {
	return Repository{
		dbService: dbService,
	}
}

func (r Repository) GetById(ctx context.Context, id ulid.ULID) (*model.User, error) {
	u := new(model.User)
	if err := r.dbService.Pool().QueryRow(ctx, getById, id).Scan(u); err != nil {
		return nil, err
	}

	return u, nil
}

func (r Repository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	u := new(model.User)
	if err := r.dbService.Pool().QueryRow(ctx, getByEmail, email).Scan(u); err != nil {
		return nil, err
	}

	return u, nil
}

func (r Repository) Create(ctx context.Context, u *model.User) (*model.User, error) {
	//TODO implement me
	panic("implement me")
}

func (r Repository) Update(ctx context.Context, u *model.User) (*model.User, error) {
	//TODO implement me
	panic("implement me")
}

func (r Repository) DeleteById(ctx context.Context, id ulid.ULID) error {
	//TODO implement me
	panic("implement me")
}
