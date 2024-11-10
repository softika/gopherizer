package account

import (
	"encoding/json"
	"net/http"

	model "github.com/softika/gopherizer/internal/account"
)

type LoginRequestMapper struct{}

func NewLoginRequestMapper() LoginRequestMapper {
	return LoginRequestMapper{}
}

func (m LoginRequestMapper) Map(r *http.Request) (model.LoginRequest, error) {
	req := new(model.LoginRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return model.LoginRequest{}, err
	}

	return *req, nil
}

type LoginResponseMapper struct{}

func NewLoginResponseMapper() LoginResponseMapper {
	return LoginResponseMapper{}
}

func (m LoginResponseMapper) Map(w http.ResponseWriter, out *model.LoginResponse) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(out)
}
