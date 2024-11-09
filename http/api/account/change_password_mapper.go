package account

import (
	"encoding/json"
	"net/http"

	model "tldw/internal/account"
)

type ChangePasswordMapper struct{}

func NewChangePasswordMapper() ChangePasswordMapper {
	return ChangePasswordMapper{}
}

func (m ChangePasswordMapper) Map(r *http.Request) (model.ChangePasswordRequest, error) {
	req := new(model.ChangePasswordRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return model.ChangePasswordRequest{}, err
	}

	return *req, nil
}

type ChangePasswordResponseMapper struct{}

func NewChangePasswordResponseMapper() ChangePasswordResponseMapper {
	return ChangePasswordResponseMapper{}
}

func (m ChangePasswordResponseMapper) Map(w http.ResponseWriter, out *model.ChangePasswordResponse) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(out)
}
