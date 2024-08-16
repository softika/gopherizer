//go:generate mockgen -source=user.go -destination=./mock/user.go -package=mock
package user

import (
	"context"
	"fmt"

	"tldw/internal/model"
	"tldw/internal/services"
)

type Repository interface {
	GetById(context.Context, string) (*model.User, error)
	GetByEmail(context.Context, string) (*model.User, error)
	Create(context.Context, *model.User) (*model.User, error)
	Update(context.Context, *model.User) (*model.User, error)
	DeleteById(context.Context, string) error
}

type Service struct {
	repo Repository
}

func NewService(r Repository) Service {
	return Service{repo: r}
}

func (s Service) GetById(ctx context.Context, id string) (*Response, error) {
	u, err := s.repo.GetById(ctx, id)
	if err != nil {
		return nil, services.NewError(
			fmt.Errorf("failed to get user by id: %w", err),
			services.ErrNotFound,
		)
	}

	res := new(Response)
	return res.fromModel(u), nil
}

func (s Service) GetByEmail(ctx context.Context, email string) (*Response, error) {
	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, services.NewError(
			fmt.Errorf("failed to get user by email: %w", err),
			services.ErrNotFound,
		)
	}

	res := new(Response)
	return res.fromModel(u), nil
}

func (s Service) Create(ctx context.Context, req CreateRequest) (*Response, error) {
	u := model.NewUser().
		WithFirstName(req.FirstName).
		WithLastName(req.LastName).
		WithEmail(req.Email).
		WithPassword(req.Password)

	created, err := s.repo.Create(ctx, u)
	if err != nil {
		return nil, services.NewError(
			fmt.Errorf("failed to create user: %w", err),
			services.ErrInternal,
		)
	}

	res := new(Response)
	return res.fromModel(created), nil
}

func (s Service) DeleteById(ctx context.Context, id string) (bool, error) {
	if err := s.repo.DeleteById(ctx, id); err != nil {
		return false, services.NewError(
			fmt.Errorf("failed to delete user by id: %w", err),
			services.ErrInternal,
		)
	}

	return true, nil
}
