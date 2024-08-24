package health

import "context"

type Repository interface {
	Health(ctx context.Context) map[string]string
}

type Request struct {
	Status string
}

type Response map[string]string

// Service is a dummy service to confirm the health of the server.
type Service struct {
	repo Repository
}

func NewService(r Repository) Service {
	return Service{
		repo: r,
	}
}

// Check respond with the health status.
func (s Service) Check(ctx context.Context, _ Request) (*Response, error) {
	res := s.repo.Health(ctx)
	response := Response(res)
	return &response, nil
}
