package services

import "errors"

type ErrorCode int

const (
	ErrInternal ErrorCode = iota
	ErrInvalidInput
	ErrForbidden
	ErrNotFound
)

type Error struct {
	Err  error
	Code ErrorCode
}

func NewError(err error, code ErrorCode) *Error {
	return &Error{
		Err:  err,
		Code: code,
	}
}

func NewErrorMessage(msg string, code ErrorCode) *Error {
	return &Error{
		Err:  errors.New(msg),
		Code: code,
	}
}

func (e *Error) Error() string {
	return e.Err.Error()
}
