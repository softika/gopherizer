package profile

import (
	"encoding/json"
	"net/http"

	svc "tldw/internal/services/profile"
)

type UpdateRequestMapper struct{}

func NewUpdateRequestMapper() UpdateRequestMapper {
	return UpdateRequestMapper{}
}

func (m UpdateRequestMapper) Map(r *http.Request) (svc.UpdateRequest, error) {
	req := new(svc.UpdateRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return svc.UpdateRequest{}, err
	}

	return *req, nil
}

type UpdateResponseMapper struct{}

func NewUpdateResponseMapper() UpdateResponseMapper {
	return UpdateResponseMapper{}
}

func (m UpdateResponseMapper) Map(w http.ResponseWriter, out *svc.Response) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(out)
}
