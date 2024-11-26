package mappers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/softika/gopherizer/internal/profile"
)

type DeleteProfileRequest struct{}

func (m DeleteProfileRequest) Map(r *http.Request) (profile.DeleteRequest, error) {
	id := r.PathValue("id")
	if id == "" {
		return profile.DeleteRequest{}, fmt.Errorf("path param id is missing")
	}

	return profile.DeleteRequest{Id: id}, nil
}

type DeleteProfileResponse struct{}

func (m DeleteProfileResponse) Map(w http.ResponseWriter, d bool) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)

	res := map[string]bool{"deleted": d}
	return json.NewEncoder(w).Encode(res)
}
