package profile

import (
	"encoding/json"
	"net/http"

	svc "tldw/internal/services/profile"
)

type CreateRequestMapper struct{}

func NewCreateRequestMapper() CreateRequestMapper {
	return CreateRequestMapper{}
}

func (m CreateRequestMapper) Map(r *http.Request) (svc.CreateRequest, error) {
	req := new(svc.CreateRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return svc.CreateRequest{}, err
	}

	return *req, nil
}

type CreateResponseMapper struct{}

func NewCreateResponseMapper() CreateResponseMapper {
	return CreateResponseMapper{}
}

func (m CreateResponseMapper) Map(w http.ResponseWriter, out *svc.Response) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(out)
}
