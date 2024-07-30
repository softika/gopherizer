package errorx

type ErrorType int

const (
	ErrInternal ErrorType = iota
	ErrInvalidInput
	ErrForbidden
	ErrNotFound
	ErrUnauthorized
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
