package repositories

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/softika/gopherizer/pkg/errorx"
)

// classify tags a driver failure with a domain error type.
//
// The repository is the only layer that knows pgx, so it is the only place that
// can tell a missing row apart from a genuine failure. Everything above works
// with errorx types instead of driver sentinels.
func classify(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return errorx.NewError(err, errorx.ErrNotFound)
	}
	return errorx.NewError(err, errorx.ErrInternal)
}

// notFound reports a row that the statement did not match.
func notFound(entity, id string) error {
	return errorx.NewError(
		fmt.Errorf("%s %q not found", entity, id),
		errorx.ErrNotFound,
	)
}
