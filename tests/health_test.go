package tests

import (
	"net/http"
	"net/http/httptest"

	"github.com/go-playground/validator/v10"

	"github.com/softika/gopherizer/api"
	healthApi "github.com/softika/gopherizer/api/health"
	healthSvc "github.com/softika/gopherizer/internal/health"
)

func (s *E2ETestSuite) TestHealthEndpoint() {
	svc := healthSvc.NewService(s.dbService)

	handler := api.NewHandler(
		healthApi.NewRequestMapper(),
		healthApi.NewResponseMapper(),
		svc.Check,
		validator.New(),
	)

	handler.Route(s.router, http.MethodGet, "/health")

	testCases := []struct {
		name     string
		wantCode int
	}{
		{
			name:     "health check",
			wantCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		tt := tc
		s.Run(tt.name, func() {
			s.T().Parallel()
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			w := httptest.NewRecorder()

			s.router.ServeHTTP(w, req)

			s.Equal(tt.wantCode, w.Code)
			s.NotEmpty(s.T(), w.Body.String())
		})
	}
}
