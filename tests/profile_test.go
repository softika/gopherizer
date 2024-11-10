package tests

import (
	"bytes"
	"encoding/json"
	userSvc "github.com/softika/gopherizer/internal/profile"
	"net/http"
	"net/http/httptest"
)

func (s *E2ETestSuite) TestCreateProfileHandler() {
	testCases := []struct {
		name     string
		path     string
		req      userSvc.CreateRequest
		wantCode int
	}{}

	for _, tc := range testCases {
		tt := tc
		s.Run(tt.name, func() {
			s.T().Parallel()

			// given
			body, err := json.Marshal(tt.req)
			s.NoError(err)

			req := httptest.NewRequest(http.MethodPost, tt.path, bytes.NewReader(body))
			w := httptest.NewRecorder()

			// when
			s.router.ServeHTTP(w, req)

			// then
			s.Equal(tt.wantCode, w.Code)
			s.NotEmpty(s.T(), w.Body.String())
		})
	}
}
