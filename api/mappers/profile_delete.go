package mappers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type DeleteProfileRequest struct{}

func (m DeleteProfileRequest) Map(r *http.Request) (string, error) {
	id := r.PathValue("id")
	if id == "" {
		return "", fmt.Errorf("path param id is missing")
	}

	return id, nil
}

type DeleteProfileResponse struct{}

func (m DeleteProfileResponse) Map(w http.ResponseWriter, d bool) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	res := map[string]bool{"deleted": d}
	return json.NewEncoder(w).Encode(res)
}
