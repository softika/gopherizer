package account

import (
	"encoding/json"
	"net/http"

	model "github.com/softika/gopherizer/internal/account"
)

type RegisterRequestMapper struct{}

func NewRegisterRequestMapper() RegisterRequestMapper {
	return RegisterRequestMapper{}
}

func (m RegisterRequestMapper) Map(r *http.Request) (model.RegisterRequest, error) {
	req := new(model.RegisterRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return model.RegisterRequest{}, err
	}

	return *req, nil
}

type RegisterResponseMapper struct{}

func NewRegisterResponseMapper() RegisterResponseMapper {
	return RegisterResponseMapper{}
}

func (m RegisterResponseMapper) Map(w http.ResponseWriter, out *model.RegisterResponse) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(out)
}
