//go:generate mockgen -source=service.go -destination=./mock/service.go -package=mock
package profile

import (
	"context"
	"fmt"

	"github.com/softika/slogging"

	"github.com/softika/gopherizer/internal"
	"github.com/softika/gopherizer/pkg/errorx"
)

type Repository interface {
	internal.Repository[Profile, string]
}

type Service struct {
	repo Repository
}

func NewService(r Repository) Service {
	return Service{repo: r}
}

func (s Service) GetById(ctx context.Context, req GetRequest) (*Response, error) {
	u, err := s.repo.GetById(ctx, req.Id)
	if err != nil {
		logger := slogging.Slogger()
		logger.ErrorContext(ctx, "failed to get profile by id", "id", req.Id, "error", err)
		return nil, errorx.NewError(
			fmt.Errorf("failed to get profile by id: %w", err),
			errorx.ErrNotFound,
		)
	}

	res := new(Response)
	return res.fromModel(u), nil
}

func (s Service) Create(ctx context.Context, req CreateRequest) (*Response, error) {
	p := New().WithFirstName(req.FirstName).WithLastName(req.LastName)

	created, err := s.repo.Create(ctx, p)
	if err != nil {
		slogging.Slogger().ErrorContext(ctx, "failed to create profile", "error", err)
		return nil, errorx.NewError(
			fmt.Errorf("failed to create profile: %w", err),
			errorx.ErrInternal,
		)
	}

	res := new(Response)
	res.fromModel(created)

	return res, nil
}

func (s Service) Update(ctx context.Context, req UpdateRequest) (*Response, error) {
	u := New().WithId(req.Id).WithFirstName(req.FirstName).WithLastName(req.LastName)

	updated, err := s.repo.Update(ctx, u)
	if err != nil {
		slogging.Slogger().ErrorContext(ctx, "failed to update profile", "error", err)
		return nil, errorx.NewError(
			fmt.Errorf("failed to update profile: %w", err),
			errorx.ErrInternal,
		)
	}

	res := new(Response)
	res.fromModel(updated)

	return res, nil
}

func (s Service) DeleteById(ctx context.Context, req DeleteRequest) (bool, error) {
	if err := s.repo.DeleteById(ctx, req.Id); err != nil {
		logger := slogging.Slogger()
		logger.ErrorContext(ctx, "failed to delete profile by id", "id", req.Id, "error", err)
		return false, errorx.NewError(
			fmt.Errorf("failed to delete profile by id: %w", err),
			errorx.ErrInternal,
		)
	}

	return true, nil
}
