package health

import (
	"encoding/json"
	"net/http"

	"github.com/softika/gopherizer/internal/health"
)

type RequestMapper struct{}

func NewRequestMapper() RequestMapper {
	return RequestMapper{}
}

func (rm RequestMapper) Map(*http.Request) (health.Request, error) {
	return health.Request{Status: "OK"}, nil
}

type ResponseMapper struct{}

func NewResponseMapper() ResponseMapper {
	return ResponseMapper{}
}

func (rm ResponseMapper) Map(w http.ResponseWriter, out *health.Response) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	return json.NewEncoder(w).Encode(out)
}
