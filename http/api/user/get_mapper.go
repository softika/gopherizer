package user

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/oklog/ulid/v2"

	svcUser "tldw/internal/services/user"
)

type GetByIdRequestMapper struct{}

func NewGetByIdRequestMapper() GetByIdRequestMapper {
	return GetByIdRequestMapper{}
}

func (g GetByIdRequestMapper) Map(r *http.Request) (ulid.ULID, error) {
	idParam := r.PathValue("id")
	id, err := ulid.Parse(idParam)
	if err != nil {
		return ulid.ULID{}, fmt.Errorf("failed to parse id: %w", err)
	}
	return id, nil
}

type GetByIdResponseMapper struct{}

func NewGetByIdResponseMapper() GetByIdResponseMapper {
	return GetByIdResponseMapper{}
}

func (g GetByIdResponseMapper) Map(w http.ResponseWriter, out *svcUser.Response) error {
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

func (g GetByEmailResponseMapper) Map(w http.ResponseWriter, out *svcUser.Response) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(out)
}
