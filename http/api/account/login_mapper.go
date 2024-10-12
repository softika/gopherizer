package account

import (
	"encoding/json"
	"net/http"

	svc "tldw/internal/services/account"
)

type LoginRequestMapper struct{}

func NewLoginRequestMapper() LoginRequestMapper {
	return LoginRequestMapper{}
}

func (m LoginRequestMapper) Map(r *http.Request) (svc.LoginRequest, error) {
	req := new(svc.LoginRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return svc.LoginRequest{}, err
	}

	return *req, nil
}

type LoginResponseMapper struct{}

func NewLoginResponseMapper() LoginResponseMapper {
	return LoginResponseMapper{}
}

func (m LoginResponseMapper) Map(w http.ResponseWriter, out *svc.LoginResponse) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(out)
}
