package api

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/softika/gopherizer/internal/health"
	"github.com/softika/gopherizer/internal/profile"
)

// Inputs and outputs are HTTP-shaped wrappers around the domain types. huma
// reads the request from the struct tags and derives the OpenAPI schema from
// the same declaration, so the document cannot drift from the code.

type healthOutput struct {
	Body *health.Response
}

type profileOutput struct {
	Body *profile.Response
}

type createProfileInput struct {
	Body profile.CreateRequest
}

type updateProfileInput struct {
	Body profile.UpdateRequest
}

// The id is constrained with `pattern` rather than `format`: JSON Schema
// treats format as an annotation validators may ignore, while pattern is always
// enforced, so a malformed id is rejected before it reaches the database.
type profileIdInput struct {
	Id string `path:"id" pattern:"^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$" doc:"Profile identifier"`
}

// registerOperations wires every endpoint onto the huma API.
//
// Each registration is the single source of truth for routing, request
// decoding, validation and the OpenAPI operation.
func registerOperations(api huma.API, s services) {
	registerHealth(api, s.health)
	registerProfile(api, s.profile)
}

func registerHealth(api huma.API, svc health.Service) {
	huma.Register(api, huma.Operation{
		OperationID: "health-live",
		Method:      http.MethodGet,
		Path:        "/health/live",
		Summary:     "Liveness probe",
		Description: "Reports whether the process is running. Touches no dependency.",
		Tags:        []string{"Health"},
	}, func(ctx context.Context, _ *struct{}) (*healthOutput, error) {
		res, err := svc.Live(ctx, health.Request{})
		if err != nil {
			return nil, apiError(ctx, err)
		}
		return &healthOutput{Body: res}, nil
	})

	readiness := func(ctx context.Context, _ *struct{}) (*healthOutput, error) {
		res, err := svc.Check(ctx, health.Request{})
		if err != nil {
			return nil, apiError(ctx, err)
		}
		return &healthOutput{Body: res}, nil
	}

	huma.Register(api, huma.Operation{
		OperationID: "health-ready",
		Method:      http.MethodGet,
		Path:        "/health/ready",
		Summary:     "Readiness probe",
		Description: "Reports whether this instance can serve traffic.",
		Tags:        []string{"Health"},
	}, readiness)

	// Retained for probes configured against the original path.
	huma.Register(api, huma.Operation{
		OperationID: "health",
		Method:      http.MethodGet,
		Path:        "/health",
		Summary:     "Readiness probe (legacy path)",
		Tags:        []string{"Health"},
		Deprecated:  true,
	}, readiness)
}

func registerProfile(api huma.API, svc profile.Service) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-profile",
		Method:        http.MethodPost,
		Path:          "/api/v1/profile",
		Summary:       "Create profile",
		Tags:          []string{"Profile"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *createProfileInput) (*profileOutput, error) {
		res, err := svc.Create(ctx, in.Body)
		if err != nil {
			return nil, apiError(ctx, err)
		}
		return &profileOutput{Body: res}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-profile",
		Method:      http.MethodPut,
		Path:        "/api/v1/profile",
		Summary:     "Update profile",
		Tags:        []string{"Profile"},
	}, func(ctx context.Context, in *updateProfileInput) (*profileOutput, error) {
		res, err := svc.Update(ctx, in.Body)
		if err != nil {
			return nil, apiError(ctx, err)
		}
		return &profileOutput{Body: res}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-profile",
		Method:      http.MethodGet,
		Path:        "/api/v1/profile/{id}",
		Summary:     "Get profile by id",
		Tags:        []string{"Profile"},
	}, func(ctx context.Context, in *profileIdInput) (*profileOutput, error) {
		res, err := svc.GetById(ctx, profile.GetRequest{Id: in.Id})
		if err != nil {
			return nil, apiError(ctx, err)
		}
		return &profileOutput{Body: res}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "delete-profile",
		Method:        http.MethodDelete,
		Path:          "/api/v1/profile/{id}",
		Summary:       "Delete profile by id",
		Tags:          []string{"Profile"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *profileIdInput) (*struct{}, error) {
		if _, err := svc.DeleteById(ctx, profile.DeleteRequest{Id: in.Id}); err != nil {
			return nil, apiError(ctx, err)
		}
		return nil, nil
	})
}
