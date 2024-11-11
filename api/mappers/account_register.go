package mappers

import (
	"encoding/json"
	"net/http"

	"github.com/softika/gopherizer/internal/account"
)

type RegisterRequest struct{}

func (m RegisterRequest) Map(r *http.Request) (account.RegisterRequest, error) {
	req := new(account.RegisterRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return account.RegisterRequest{}, err
	}

	return *req, nil
}

type RegisterResponse struct{}

func (m RegisterResponse) Map(w http.ResponseWriter, out *account.RegisterResponse) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(out)
}
