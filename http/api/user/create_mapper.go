package user

import (
	"encoding/json"
	"net/http"

	userSvc "tldw/internal/services/user"
)

type CreateRequestMapper struct{}

func NewCreateRequestMapper() CreateRequestMapper {
	return CreateRequestMapper{}
}

func (m CreateRequestMapper) Map(r *http.Request) (userSvc.CreateRequest, error) {
	req := new(userSvc.CreateRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return userSvc.CreateRequest{}, err
	}

	return *req, nil
}

type CreateResponseMapper struct{}

func NewCreateResponseMapper() CreateResponseMapper {
	return CreateResponseMapper{}
}

func (m CreateResponseMapper) Map(w http.ResponseWriter, out *userSvc.Response) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(out)
}
