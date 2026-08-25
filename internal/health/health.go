package health

import (
	"context"
	"errors"
	"log/slog"

	"github.com/softika/gopherizer/pkg/errorx"
)

// statusUp is the value the database service reports when it is reachable.
const statusUp = "up"

type Repository interface {
	Health(ctx context.Context) map[string]string
}

type Request struct {
	Status string
}

// Response is deliberately minimal. Connection counts and timings describe the
// infrastructure, so they are logged rather than served to callers.
type Response struct {
	Status string `json:"status"`
}

// Service reports whether the server and its dependencies can serve traffic.
type Service struct {
	repo Repository
}

func NewService(r Repository) Service {
	return Service{
		repo: r,
	}
}

// Live reports process liveness and deliberately touches no dependency.
//
// Liveness answers "should this process be restarted". Checking the database
// here would let a brief outage cause an orchestrator to kill healthy pods,
// turning a recoverable blip into a restart storm.
func (s Service) Live(_ context.Context, _ Request) (*Response, error) {
	return &Response{Status: statusUp}, nil
}

// Check reports readiness: whether this instance can serve traffic right now.
//
// A failing dependency returns ErrUnavailable so the caller sees 503 and load
// balancers stop routing here, instead of a 200 that hides the outage.
func (s Service) Check(ctx context.Context, _ Request) (*Response, error) {
	stats := s.repo.Health(ctx)

	if stats["status"] != statusUp {
		// The detail is useful to operators, not to callers.
		slog.ErrorContext(ctx, "readiness check failed", "stats", stats)

		return nil, errorx.NewError(
			errors.New("dependency unavailable"),
			errorx.ErrUnavailable,
		)
	}

	slog.DebugContext(ctx, "readiness check passed", "stats", stats)

	return &Response{Status: statusUp}, nil
}
