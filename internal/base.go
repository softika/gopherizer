package internal

import (
	"context"
	"time"
)

// Base is a base model for all model entities.
type Base struct {
	Id        string    `db:"id,pk"` // uuid
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

// PageRequest is a pagination request.
type PageRequest struct {
	Limit  int
	Offset int
}

// DefaultPageRequest returns default page request.
func DefaultPageRequest() PageRequest {
	return PageRequest{Limit: 10, Offset: 0}
}

// Page is a generic pagination response.
type Page[T any] struct {
	TotalPages int
	TotalItems int
	Items      []T
}

func EmptyPage[T any]() Page[T] {
	return Page[T]{
		TotalPages: 0,
		TotalItems: 0,
		Items:      []T{},
	}
}

// Repository is embeddable generic repository.
type Repository[T any, ID any] interface {
	GetById(context.Context, ID) (*T, error)
	Create(context.Context, *T) (*T, error)
	Update(context.Context, *T) (*T, error)
	DeleteById(context.Context, ID) error
}
