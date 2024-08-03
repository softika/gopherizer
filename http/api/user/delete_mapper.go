package user

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/oklog/ulid/v2"
)

type DeleteByIdRequestMapper struct{}

func NewDeleteByIdRequestMapper() DeleteByIdRequestMapper {
	return DeleteByIdRequestMapper{}
}

func (m DeleteByIdRequestMapper) Map(r *http.Request) (ulid.ULID, error) {
	idParam := r.PathValue("id")
	id, err := ulid.Parse(idParam)
	if err != nil {
		return ulid.ULID{}, fmt.Errorf("failed to parse id: %w", err)
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
