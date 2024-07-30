package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"tldw/internal/services"
)

type Error struct {
	Code     int    `json:"-"`
	Message  string `json:"message"`
	Internal error  `json:"-"` // Stores the error returned by an external dependency
}

func newError(code int, message string, internal error) Error {
	return Error{
		Code:     code,
		Message:  message,
		Internal: internal,
	}
}

func newServiceError(internal error) Error {
	code := http.StatusInternalServerError
	// Check if the error is a service error
	// and set the appropriate HTTP status code
	var errService *services.Error
	if errors.As(internal, &errService) {
		switch errService.Code {
		case services.ErrInvalidInput:
			code = http.StatusBadRequest
		case services.ErrForbidden:
			code = http.StatusForbidden
		case services.ErrNotFound:
			code = http.StatusNotFound
		default:
			code = http.StatusInternalServerError
		}
	}

	return Error{
		Code:     code,
		Message:  internal.Error(),
		Internal: internal,
	}
}

func (e Error) Error() string {
	jsonErr, err := json.Marshal(e)
	if err != nil {
		return fmt.Sprintf(`{"message":"%s"}`, e.Message)
	}
	return string(jsonErr)
}
