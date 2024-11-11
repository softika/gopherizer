package tests

import (
	"net/http"
	"net/http/httptest"
)

func (s *E2ETestSuite) TestHealthEndpoint() {
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
