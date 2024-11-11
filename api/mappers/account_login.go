package mappers

import (
	"encoding/json"
	"net/http"

	"github.com/softika/gopherizer/internal/account"
)

type LoginRequest struct{}

func (m LoginRequest) Map(r *http.Request) (account.LoginRequest, error) {
	req := new(account.LoginRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return account.LoginRequest{}, err
	}

	return *req, nil
}

type LoginResponse struct{}

func (m LoginResponse) Map(w http.ResponseWriter, out *account.LoginResponse) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(out)
}
