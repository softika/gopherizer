package mappers

import (
	"encoding/json"
	"net/http"

	"github.com/softika/gopherizer/internal/profile"
)

type UpdateProfileRequest struct{}

func (m UpdateProfileRequest) Map(r *http.Request) (profile.UpdateRequest, error) {
	req := new(profile.UpdateRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return profile.UpdateRequest{}, err
	}

	return *req, nil
}

type UpdateProfileResponse struct{}

func (m UpdateProfileResponse) Map(w http.ResponseWriter, out *profile.Response) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(out)
}
