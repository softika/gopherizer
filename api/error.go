package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/softika/gopherizer/pkg/errorx"
)

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Cause   error  `json:"cause"`
}

func newError(code int, message string, internal error) Error {
	return Error{
		Code:    code,
		Message: message,
		Cause:   internal,
	}
}

func newServiceError(err error) Error {
	code := http.StatusInternalServerError
	// Check if the error is a service error
	// and set the appropriate HTTP status code
	var errService *errorx.Error
	if errors.As(err, &errService) {
		switch errService.Type {
		case errorx.ErrInvalidInput:
			code = http.StatusBadRequest
		case errorx.ErrForbidden:
			code = http.StatusForbidden
		case errorx.ErrNotFound:
			code = http.StatusNotFound
		default:
			code = http.StatusInternalServerError
		}
	}

	return Error{
		Code:    code,
		Message: err.Error(),
		Cause:   err,
	}
}

func (e Error) Error() string {
	jsonErr, err := json.Marshal(e)
	if err != nil {
		b := strings.Builder{}
		b.WriteString(`{"message":"`)
		b.WriteString(e.Message)
		b.WriteString(`","code":`)
		b.WriteString(strconv.Itoa(e.Code))
		b.WriteString(`,"cause":"`)
		b.WriteString(e.Cause.Error())
		b.WriteString(`}`)
		return b.String()
	}
	return string(jsonErr)
}
