package user

import (
	"encoding/json"
	"net/http"

	userSvc "tldw/internal/services/user"
)

type UpdateRequestMapper struct{}

func NewUpdateRequestMapper() UpdateRequestMapper {
	return UpdateRequestMapper{}
}

func (m UpdateRequestMapper) Map(r *http.Request) (userSvc.UpdateRequest, error) {
	req := new(userSvc.UpdateRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return userSvc.UpdateRequest{}, err
	}

	return *req, nil
}

type UpdateResponseMapper struct{}

func NewUpdateResponseMapper() UpdateResponseMapper {
	return UpdateResponseMapper{}
}

func (m UpdateResponseMapper) Map(w http.ResponseWriter, out *userSvc.Response) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(out)
}
