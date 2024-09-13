package account

import (
	"encoding/json"
	"net/http"

	svc "tldw/internal/services/account"
)

type RegisterRequestMapper struct{}

func NewRegisterRequestMapper() RegisterRequestMapper {
	return RegisterRequestMapper{}
}

func (m RegisterRequestMapper) Map(r *http.Request) (svc.RegisterRequest, error) {
	req := new(svc.RegisterRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return svc.RegisterRequest{}, err
	}

	return *req, nil
}

type RegisterResponseMapper struct{}

func NewRegisterResponseMapper() RegisterResponseMapper {
	return RegisterResponseMapper{}
}

func (m RegisterResponseMapper) Map(w http.ResponseWriter, out *svc.RegisterResponse) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(out)
}
