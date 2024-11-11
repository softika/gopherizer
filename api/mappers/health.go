package mappers

import (
	"encoding/json"
	"net/http"

	"github.com/softika/gopherizer/internal/health"
)

type HealthRequest struct{}

func (rm HealthRequest) Map(*http.Request) (health.Request, error) {
	return health.Request{Status: "OK"}, nil
}

type HealthResponse struct{}

func (rm HealthResponse) Map(w http.ResponseWriter, out *health.Response) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(out)
}
