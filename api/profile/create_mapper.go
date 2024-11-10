package profile

import (
	"encoding/json"
	"net/http"

	model "github.com/softika/gopherizer/internal/profile"
)

type CreateRequestMapper struct{}

func NewCreateRequestMapper() CreateRequestMapper {
	return CreateRequestMapper{}
}

func (m CreateRequestMapper) Map(r *http.Request) (model.CreateRequest, error) {
	req := new(model.CreateRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return model.CreateRequest{}, err
	}

	return *req, nil
}

type CreateResponseMapper struct{}

func NewCreateResponseMapper() CreateResponseMapper {
	return CreateResponseMapper{}
}

func (m CreateResponseMapper) Map(w http.ResponseWriter, out *model.Response) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(out)
}
