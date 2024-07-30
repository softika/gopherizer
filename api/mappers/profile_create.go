package mappers

import (
	"encoding/json"
	"net/http"

	"github.com/softika/gopherizer/internal/profile"
)

type CreateProfileRequest struct{}

func (m CreateProfileRequest) Map(r *http.Request) (profile.CreateRequest, error) {
	req := new(profile.CreateRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return profile.CreateRequest{}, err
	}

	return *req, nil
}

type CreateProfileResponse struct{}

func (m CreateProfileResponse) Map(w http.ResponseWriter, out *profile.Response) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(out)
}
