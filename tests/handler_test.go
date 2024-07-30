package tests

import (
	"testing"
	health2 "tldw/internal/services/health"

	"github.com/go-playground/validator/v10"

	"tldw/http/api"
	"tldw/http/api/health"
)

func TestHealthHandler(t *testing.T) {

	_ = api.NewHandler(
		api.NewRouter("test", "test"),
		health.NewRequestMapper(),
		health.NewResponseMapper(),
		health2.NewService().Check,
		validator.New(),
	)

}
