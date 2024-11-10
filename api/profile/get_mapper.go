package profile

import (
	"encoding/json"
	"fmt"
	"net/http"

	model "github.com/softika/gopherizer/internal/profile"
)

type GetByIdRequestMapper struct{}

func NewGetByIdRequestMapper() GetByIdRequestMapper {
	return GetByIdRequestMapper{}
}

func (g GetByIdRequestMapper) Map(r *http.Request) (string, error) {
	id := r.PathValue("id")
	if id == "" {
		return "", fmt.Errorf("path param id is missing")
	}
	return id, nil
}

type GetByIdResponseMapper struct{}

func NewGetByIdResponseMapper() GetByIdResponseMapper {
	return GetByIdResponseMapper{}
}

func (g GetByIdResponseMapper) Map(w http.ResponseWriter, out *model.Response) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(out)
}

type GetByEmailRequestMapper struct{}

func NewGetByEmailRequestMapper() GetByEmailRequestMapper {
	return GetByEmailRequestMapper{}
}

func (g GetByEmailRequestMapper) Map(r *http.Request) (string, error) {
	email := r.URL.Query().Get("email")
	if email == "" {
		return "", fmt.Errorf("email is required")
	}
	return email, nil
}

type GetByEmailResponseMapper struct{}

func NewGetByEmailResponseMapper() GetByEmailResponseMapper {
	return GetByEmailResponseMapper{}
}

func (g GetByEmailResponseMapper) Map(w http.ResponseWriter, out *model.Response) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(out)
}
