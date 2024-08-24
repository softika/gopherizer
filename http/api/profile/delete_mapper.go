package profile

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type DeleteByIdRequestMapper struct{}

func NewDeleteByIdRequestMapper() DeleteByIdRequestMapper {
	return DeleteByIdRequestMapper{}
}

func (m DeleteByIdRequestMapper) Map(r *http.Request) (string, error) {
	id := r.PathValue("id")
	if id == "" {
		return "", fmt.Errorf("path param id is missing")
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
