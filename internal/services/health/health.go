package health

import "context"

type Request struct {
	Status string
}

type Response struct {
	Status string `json:"status"`
}

// Service is a dummy service to confirm the health of the server.
type Service struct{}

func NewService() Service {
	return Service{}
}

// Check is a dummy function that takes the request status and puts it in a response.
func (hs Service) Check(_ context.Context, in Request) (*Response, error) {
	return &Response{Status: in.Status}, nil
}
