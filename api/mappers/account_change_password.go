package mappers

import (
	"encoding/json"
	"net/http"

	"github.com/softika/gopherizer/internal/account"
)

type ChangePasswordRequest struct{}

func (m ChangePasswordRequest) Map(r *http.Request) (account.ChangePasswordRequest, error) {
	req := new(account.ChangePasswordRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return account.ChangePasswordRequest{}, err
	}

	return *req, nil
}

type ChangePasswordResponse struct{}

func (m ChangePasswordResponse) Map(w http.ResponseWriter, out *account.ChangePasswordResponse) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(out)
}
