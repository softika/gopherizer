package user

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	svcUser "tldw/internal/services/user"
)

type GetByIdRequestMapper struct{}

func NewGetByIdRequestMapper() GetByIdRequestMapper {
	return GetByIdRequestMapper{}
}

func (g GetByIdRequestMapper) Map(r *http.Request) (uuid.UUID, error) {
	idParam := r.PathValue("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to parse id: %w", err)
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
