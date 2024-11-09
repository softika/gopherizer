package profile

import (
	"encoding/json"
	"net/http"

	model "tldw/internal/profile"
)

type UpdateRequestMapper struct{}

func NewUpdateRequestMapper() UpdateRequestMapper {
	return UpdateRequestMapper{}
}

func (m UpdateRequestMapper) Map(r *http.Request) (model.UpdateRequest, error) {
	req := new(model.UpdateRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return model.UpdateRequest{}, err
	}

	return *req, nil
}

type UpdateResponseMapper struct{}

func NewUpdateResponseMapper() UpdateResponseMapper {
	return UpdateResponseMapper{}
}

func (m UpdateResponseMapper) Map(w http.ResponseWriter, out *model.Response) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(out)
}
