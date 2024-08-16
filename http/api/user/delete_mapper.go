package user

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

type DeleteByIdRequestMapper struct{}

func NewDeleteByIdRequestMapper() DeleteByIdRequestMapper {
	return DeleteByIdRequestMapper{}
}

func (m DeleteByIdRequestMapper) Map(r *http.Request) (uuid.UUID, error) {
	idParam := r.PathValue("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to parse id: %w", err)
	}
	return id, nil
}

type DeleteByIdResponseMapper struct{}

func NewDeleteByIdResponseMapper() DeleteByIdResponseMapper {
	return DeleteByIdResponseMapper{}
}

func (m DeleteByIdResponseMapper) Map(w http.ResponseWriter, d bool) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	res := map[string]bool{"deleted": d}
	return json.NewEncoder(w).Encode(res)
}
