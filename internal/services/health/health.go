package health

import "context"

type Repository interface {
	Health() map[string]string
}

type Request struct {
	Status string
}

type Response struct {
	Status map[string]string `json:"status"`
}

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
func (s Service) Check(_ context.Context, _ Request) (*Response, error) {
	res := s.repo.Health()
	return &Response{Status: res}, nil
}
