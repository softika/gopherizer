package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/softika/gopherizer/pkg/errorx"
)

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Cause   error  `json:"-"`
}

func newError(code int, message string, internal error) Error {
	return Error{
		Code:    code,
		Message: message,
		Cause:   internal,
	}
}

// clientError is the client-facing rendering of a domain error type.
// Messages are deliberately generic: internal detail stays in Cause, which is
// logged server-side and never serialized.
type clientError struct {
	code    int
	message string
}

var clientErrors = map[errorx.ErrorType]clientError{
	errorx.ErrInvalidInput: {http.StatusBadRequest, "invalid request"},
	errorx.ErrUnauthorized: {http.StatusUnauthorized, "unauthorized"},
	errorx.ErrForbidden:    {http.StatusForbidden, "forbidden"},
	errorx.ErrNotFound:     {http.StatusNotFound, "not found"},
	errorx.ErrInternal:     {http.StatusInternalServerError, "internal server error"},
	errorx.ErrUnavailable:  {http.StatusServiceUnavailable, "service unavailable"},
}

// newServiceError converts a service failure into a safe API error.
func newServiceError(err error) Error {
	ce, ok := clientErrors[errorx.TypeOf(err)]
	if !ok {
		ce = clientErrors[errorx.ErrInternal]
	}

	return Error{
		Code:    ce.code,
		Message: ce.message,
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
		b.WriteString(`}`)
		return b.String()
	}
	return string(jsonErr)
}
