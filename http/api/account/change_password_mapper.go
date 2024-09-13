package account

import (
	"encoding/json"
	"net/http"

	svc "tldw/internal/services/account"
)

type ChangePasswordMapper struct{}

func NewChangePasswordMapper() ChangePasswordMapper {
	return ChangePasswordMapper{}
}

func (m ChangePasswordMapper) Map(r *http.Request) (svc.ChangePasswordRequest, error) {
	req := new(svc.ChangePasswordRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return svc.ChangePasswordRequest{}, err
	}

	return *req, nil
}

type ChangePasswordResponseMapper struct{}

func NewChangePasswordResponseMapper() ChangePasswordResponseMapper {
	return ChangePasswordResponseMapper{}
}

func (m ChangePasswordResponseMapper) Map(w http.ResponseWriter, out *svc.ChangePasswordResponse) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(out)
}
