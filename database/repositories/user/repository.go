package user

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/oklog/ulid/v2"

	"tldw/database"
	"tldw/internal/model"
)

var (
	//go:embed sql/get_by_id.sql
	getById string
	//go:embed sql/get_by_email.sql
	getByEmail string
	//go:embed sql/create.sql
	createUser string
	//go:embed sql/update.sql
	updateUser string
	//go:embed sql/delete_by_id.sql
	deleteById string
)

type Repository struct {
	dbService database.Service
}

func NewRepository(dbService database.Service) Repository {
	return Repository{dbService: dbService}
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
	err := r.dbService.Pool().QueryRow(ctx, createUser,
		u.FirstName, // $1
		u.LastName,  // $2
		u.Email,     // $3
		u.Password,  // $4
		u.Enabled,   // $5
	).Scan(u)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return u, nil
}

func (r Repository) Update(ctx context.Context, u *model.User) (*model.User, error) {
	err := r.dbService.Pool().QueryRow(ctx, updateUser,
		u.FirstName, // $1
		u.LastName,  // $2
		u.Email,     // $3
		u.Password,  // $4
		u.Enabled,   // $5
		u.Id,        // $6
	).Scan(u)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}
	return u, nil
}

func (r Repository) DeleteById(ctx context.Context, id ulid.ULID) error {
	_, err := r.dbService.Pool().Exec(ctx, deleteById, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}
