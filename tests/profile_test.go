package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/softika/gopherizer/internal/profile"
)

func (s *E2ETestSuite) TestCreateProfileHandler() {
	cases := []struct {
		name     string
		req      profile.CreateRequest
		wantCode int
	}{
		{
			name: "create profile",
			req: profile.CreateRequest{
				FirstName: "John",
				LastName:  "Snow",
			},
			wantCode: http.StatusCreated,
		},
		{
			name:     "create profile with empty request",
			req:      profile.CreateRequest{},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "create profile with empty first name",
			req: profile.CreateRequest{
				FirstName: "",
				LastName:  "Snow",
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "create profile with empty last name",
			req: profile.CreateRequest{
				FirstName: "John",
				LastName:  "",
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "create profile with long first name",
			req: profile.CreateRequest{
				FirstName: "John John John John John John John John John John John John John John John John John John John John John",
				LastName:  "Snow",
			},
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		tt := tc
		s.Run(tt.name, func() {
			s.T().Parallel()

			// given
			body, err := json.Marshal(tt.req)
			s.NoError(err)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/profile", bytes.NewReader(body))
			w := httptest.NewRecorder()

			// when
			s.router.ServeHTTP(w, req)

			// then
			s.Equal(tt.wantCode, w.Code)
			s.NotEmpty(s.T(), w.Body.String())

			if w.Code != http.StatusCreated {
				return
			}

			var res profile.Response
			err = json.Unmarshal(w.Body.Bytes(), &res)

			s.NoError(err)
			s.NotEmpty(res.Id)
			s.NotEmpty(res.CreatedAt)
			s.NotEmpty(res.UpdatedAt)
			s.Equal(tt.req.FirstName, res.FirstName)
			s.Equal(tt.req.LastName, res.LastName)
		})
	}
}

func (s *E2ETestSuite) TestGetProfileHandler() {
	cases := []struct {
		name     string
		id       string
		wantCode int
	}{
		{
			name:     "get profile by id",
			id:       "0dd35f9a-0d20-41f1-80c2-d7993e313fb4", //John Doe
			wantCode: http.StatusOK,
		},
		{
			name:     "get profile by invalid id",
			id:       "invalid",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "get profile by non-existent id",
			id:       "e72e6527-6496-43ee-961f-e9d2b97bbdf3", // non-existent
			wantCode: http.StatusNotFound,
		},
		{
			name:     "get profile by empty id",
			id:       "",
			wantCode: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range cases {
		tt := tc
		s.Run(tt.name, func() {
			s.T().Parallel()

			// given
			req := httptest.NewRequest(http.MethodGet, "/api/v1/profile/"+tt.id, nil)
			w := httptest.NewRecorder()

			// when
			s.router.ServeHTTP(w, req)

			// then
			s.Equal(tt.wantCode, w.Code)
			s.NotEmpty(s.T(), w.Body.String())

			if w.Code != http.StatusOK {
				return
			}

			var res profile.Response
			err := json.Unmarshal(w.Body.Bytes(), &res)

			s.NoError(err)
			s.NotEmpty(res.Id)
			s.NotEmpty(res.CreatedAt)
			s.NotEmpty(res.UpdatedAt)
		})
	}
}

func (s *E2ETestSuite) TestUpdateProfileHandler() {
	cases := []struct {
		name     string
		req      profile.UpdateRequest
		wantCode int
	}{
		{
			name: "update profile",
			req: profile.UpdateRequest{
				Id:        "0dd35f9a-0d20-41f1-80c2-d7993e313fb6", // Alice Wonderland
				FirstName: "Jane",
				LastName:  "Doe",
			},
			wantCode: http.StatusOK,
		},
		{
			name:     "update profile with empty request",
			req:      profile.UpdateRequest{},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "update profile with empty first name",
			req: profile.UpdateRequest{
				Id:        "0dd35f9a-0d20-41f1-80c2-d7993e313fb6", // Alice Wonderland
				FirstName: "",
				LastName:  "Doe",
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "update profile with empty last name",
			req: profile.UpdateRequest{
				Id:        "0dd35f9a-0d20-41f1-80c2-d7993e313fb6", // Alice Wonderland
				FirstName: "Jane",
				LastName:  "",
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "update profile with long first name",
			req: profile.UpdateRequest{
				Id:        "0dd35f9a-0d20-41f1-80c2-d7993e313fb6", // Alice Wonderland
				FirstName: "John John John John John John John John John John John John John John John John John John John John",
				LastName:  "Doe",
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "update profile with invalid id",
			req: profile.UpdateRequest{
				Id:        "invalid",
				FirstName: "Jane",
				LastName:  "Doe",
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "update profile with non-existent id",
			req: profile.UpdateRequest{
				Id:        "999",
				FirstName: "Jane",
				LastName:  "Doe",
			},
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		tt := tc
		s.Run(tt.name, func() {
			s.T().Parallel()

			// given
			body, err := json.Marshal(tt.req)
			s.NoError(err)

			req := httptest.NewRequest(http.MethodPut, "/api/v1/profile", bytes.NewReader(body))
			w := httptest.NewRecorder()

			// when
			s.router.ServeHTTP(w, req)

			// then
			s.Equal(tt.wantCode, w.Code)
			s.NotEmpty(s.T(), w.Body.String())

			if w.Code != http.StatusOK {
				return
			}

			var res profile.Response
			err = json.Unmarshal(w.Body.Bytes(), &res)

			s.NoError(err)
			s.Equal(tt.req.Id, res.Id)
			s.Equal(tt.req.FirstName, res.FirstName)
			s.Equal(tt.req.LastName, res.LastName)
		})
	}
}

func (s *E2ETestSuite) TestDeleteProfileHandler() {
	cases := []struct {
		name     string
		id       string
		wantCode int
	}{
		{
			name:     "delete profile by id",
			id:       "0dd35f9a-0d20-41f1-80c2-d7993e313fb7", // Bob Builder
			wantCode: http.StatusNoContent,
		},
		{
			name:     "delete profile by invalid id",
			id:       "invalid",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "delete profile by non-existent id",
			id:       "999",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "delete profile by empty id",
			id:       "",
			wantCode: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range cases {
		tt := tc
		s.Run(tt.name, func() {
			s.T().Parallel()

			// given
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/profile/"+tt.id, nil)
			w := httptest.NewRecorder()

			// when
			s.router.ServeHTTP(w, req)

			// then
			s.Equal(tt.wantCode, w.Code)

			if w.Code != http.StatusNoContent {
				return
			}

			// check if the profile is deleted
			req = httptest.NewRequest(http.MethodGet, "/api/v1/profile/"+tt.id, nil)
			w = httptest.NewRecorder()
			s.router.ServeHTTP(w, req)
			s.Equal(http.StatusNotFound, w.Code)
		})
	}
}
