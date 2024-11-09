package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	account2 "tldw/internal/account"
)

func (s *E2ETestSuite) TestRegisterAccount() {
	tests := []struct {
		name     string
		input    account2.RegisterRequest
		wantCode int
	}{
		{
			name: "valid request",
			input: account2.RegisterRequest{
				Email:    "account1@test.com",
				Password: "Password1234!",
			},
			wantCode: http.StatusCreated,
		},
		{
			name: "invalid email",
			input: account2.RegisterRequest{
				Email:    "account1",
				Password: "Password1234!",
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "invalid password",
			input: account2.RegisterRequest{
				Email:    "account1@test.com",
				Password: "Pass", // too short
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "empty request",
			input:    account2.RegisterRequest{},
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		tt := tc
		s.T().Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reqBody, err := json.Marshal(tt.input)
			s.NoError(err)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/account/register", bytes.NewReader(reqBody))
			w := httptest.NewRecorder()

			s.router.ServeHTTP(w, req)

			s.Equal(tt.wantCode, w.Code)

			if tt.wantCode != http.StatusCreated {
				return
			}

			var resp account2.RegisterResponse
			err = json.NewDecoder(w.Body).Decode(&resp)
			s.NoError(err)

			s.NotEmpty(resp.AccountId)
		})
	}
}

func (s *E2ETestSuite) TestLoginAccount() {
	tests := []struct {
		name     string
		input    account2.LoginRequest
		wantCode int
	}{
		{
			name: "valid request",
			input: account2.LoginRequest{
				Email:    "john@mail.com",
				Password: "password",
			},
			wantCode: http.StatusOK,
		},
		{
			name: "invalid email",
			input: account2.LoginRequest{
				Email:    "john",
				Password: "password",
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name: "invalid password",
			input: account2.LoginRequest{
				Email:    "john@mail.com",
				Password: "invalid",
			},
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "empty request",
			input:    account2.LoginRequest{},
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		tt := tc
		s.T().Run(tt.name, func(t *testing.T) {
			t.Parallel()

			reqBody, err := json.Marshal(tt.input)
			s.NoError(err)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/account/login", bytes.NewReader(reqBody))
			w := httptest.NewRecorder()

			s.router.ServeHTTP(w, req)

			s.Equal(tt.wantCode, w.Code)

			if tt.wantCode != http.StatusOK {
				return
			}

			var resp account2.LoginResponse
			err = json.NewDecoder(w.Body).Decode(&resp)
			s.NoError(err)

			s.NotEmpty(resp.Token)
		})
	}
}
