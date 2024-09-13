package account

import (
	"encoding/json"
	"net/http"

	svc "tldw/internal/services/account"
)

type LoginRequestMap struct{}

func NewLoginRequestMap() LoginRequestMap {
	return LoginRequestMap{}
}

func (m LoginRequestMap) Map(r *http.Request) (svc.LoginRequest, error) {
	req := new(svc.LoginRequest)
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return svc.LoginRequest{}, err
	}

	return *req, nil
}

type LoginResponseMap struct{}

func NewLoginResponseMap() LoginResponseMap {
	return LoginResponseMap{}
}

func (m LoginResponseMap) Map(w http.ResponseWriter, out *svc.LoginResponse) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(out)
}
