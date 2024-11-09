package internal

import "context"

// Repository represents generic transactional repository.
type Repository[T any, ID any] interface {
	GetById(context.Context, ID) (*T, error)
	Create(context.Context, *T) (*T, error)
	Update(context.Context, *T) (*T, error)
	DeleteById(context.Context, ID) error
}
