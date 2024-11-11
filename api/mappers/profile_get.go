package mappers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/softika/gopherizer/internal/profile"
)

type GetProfileByIdRequest struct{}

func (g GetProfileByIdRequest) Map(r *http.Request) (string, error) {
	id := r.PathValue("id")
	if id == "" {
		return "", fmt.Errorf("path param id is missing")
	}
	return id, nil
}

type GetProfileByEmailRequest struct{}

func (g GetProfileByEmailRequest) Map(r *http.Request) (string, error) {
	email := r.URL.Query().Get("email")
	if email == "" {
		return "", fmt.Errorf("email is required")
	}
	return email, nil
}

type GetProfileResponse struct{}

func (g GetProfileResponse) Map(w http.ResponseWriter, out *profile.Response) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(out)
}
