package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Decoder builds a service input from an http request.
type Decoder[In any] func(*http.Request) (In, error)

// Encoder writes a service output to the response.
type Encoder[Out any] func(http.ResponseWriter, Out) error

// The decoders and encoders below cover every shape the API currently needs.
// They are written once and instantiated per endpoint, so adding a route costs
// a line of wiring rather than a pair of new types.

// JSONBody decodes the request body into In.
func JSONBody[In any](r *http.Request) (In, error) {
	var in In

	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		return in, fmt.Errorf("invalid json body: %w", err)
	}

	return in, nil
}

// PathParam builds In from a single URL path parameter.
func PathParam[In any](name string, build func(string) In) Decoder[In] {
	return func(r *http.Request) (In, error) {
		value := r.PathValue(name)
		if value == "" {
			var zero In
			return zero, fmt.Errorf("path param %q is missing", name)
		}

		return build(value), nil
	}
}

// Static supplies a fixed input for endpoints that read nothing off the request.
func Static[In any](in In) Decoder[In] {
	return func(*http.Request) (In, error) {
		return in, nil
	}
}

// JSON writes out as a JSON body with the given status code.
func JSON[Out any](status int) Encoder[Out] {
	return func(w http.ResponseWriter, out Out) error {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)

		return json.NewEncoder(w).Encode(out)
	}
}

// NoContent replies 204 with no body.
//
// A 204 response must not carry one; net/http discards anything written after
// it, so encoding a payload here would be a silent no-op.
func NoContent[Out any](w http.ResponseWriter, _ Out) error {
	w.WriteHeader(http.StatusNoContent)

	return nil
}
