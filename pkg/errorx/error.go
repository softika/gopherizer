package errorx

import "errors"

type ErrorType int

const (
	ErrInternal ErrorType = iota
	ErrInvalidInput
	ErrForbidden
	ErrNotFound
	ErrUnauthorized
	// ErrUnavailable marks a dependency failure: the request is valid but the
	// service cannot serve it right now.
	ErrUnavailable
)

type Error struct {
	Err  error
	Type ErrorType
}

func NewError(err error, code ErrorType) *Error {
	return &Error{
		Err:  err,
		Type: code,
	}
}

func (e *Error) Error() string {
	return e.Err.Error()
}

// Unwrap exposes the wrapped error so errors.Is and errors.As keep working
// through the domain error.
func (e *Error) Unwrap() error {
	return e.Err
}

// TypeOf reports the ErrorType carried anywhere in err's chain.
// Errors that carry no type are treated as internal failures.
func TypeOf(err error) ErrorType {
	var typed *Error
	if errors.As(err, &typed) {
		return typed.Type
	}
	return ErrInternal
}
