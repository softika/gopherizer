package resolver

import (
	"errors"
	"time"
)

// Resolver is used to resolve the result of an asynchronous operation
type Resolver[T any] struct {
	ResultChan chan T
	ErrorChan  chan error

	timeout time.Duration
}

// New creates a new Resolver
func New[T any](timeout time.Duration) Resolver[T] {
	return Resolver[T]{
		ResultChan: make(chan T, 1),
		ErrorChan:  make(chan error, 1),
		timeout:    timeout,
	}
}

// Get waits for the result of the asynchronous operation and returns it
func (r Resolver[T]) Get() (T, error) {
	var zeroValue T

	select {
	case result := <-r.ResultChan:
		return result, nil
	case err := <-r.ErrorChan:
		return zeroValue, err
	case <-time.After(r.timeout):
		return zeroValue, errors.New("resolver get operation timed out")
	}
}

// Close closes the Resolver channels
func (r Resolver[T]) Close() {
	select {
	case <-r.ResultChan:
	default:
		close(r.ResultChan)
	}

	select {
	case <-r.ErrorChan:
	default:
		close(r.ErrorChan)
	}
}
